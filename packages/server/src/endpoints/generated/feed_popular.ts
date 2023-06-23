export const feed_popular = {
  "fieldName": "popular",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "popular",
        "loc": {
          "start": 8378,
          "end": 8385
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8386,
              "end": 8391
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8394,
                "end": 8399
              }
            },
            "loc": {
              "start": 8393,
              "end": 8399
            }
          },
          "loc": {
            "start": 8386,
            "end": 8399
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apis",
              "loc": {
                "start": 8407,
                "end": 8411
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Api_list",
                    "loc": {
                      "start": 8425,
                      "end": 8433
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8422,
                    "end": 8433
                  }
                }
              ],
              "loc": {
                "start": 8412,
                "end": 8439
              }
            },
            "loc": {
              "start": 8407,
              "end": 8439
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notes",
              "loc": {
                "start": 8444,
                "end": 8449
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_list",
                    "loc": {
                      "start": 8463,
                      "end": 8472
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8460,
                    "end": 8472
                  }
                }
              ],
              "loc": {
                "start": 8450,
                "end": 8478
              }
            },
            "loc": {
              "start": 8444,
              "end": 8478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organizations",
              "loc": {
                "start": 8483,
                "end": 8496
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_list",
                    "loc": {
                      "start": 8510,
                      "end": 8527
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8507,
                    "end": 8527
                  }
                }
              ],
              "loc": {
                "start": 8497,
                "end": 8533
              }
            },
            "loc": {
              "start": 8483,
              "end": 8533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projects",
              "loc": {
                "start": 8538,
                "end": 8546
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Project_list",
                    "loc": {
                      "start": 8560,
                      "end": 8572
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8557,
                    "end": 8572
                  }
                }
              ],
              "loc": {
                "start": 8547,
                "end": 8578
              }
            },
            "loc": {
              "start": 8538,
              "end": 8578
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questions",
              "loc": {
                "start": 8583,
                "end": 8592
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Question_list",
                    "loc": {
                      "start": 8606,
                      "end": 8619
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8603,
                    "end": 8619
                  }
                }
              ],
              "loc": {
                "start": 8593,
                "end": 8625
              }
            },
            "loc": {
              "start": 8583,
              "end": 8625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routines",
              "loc": {
                "start": 8630,
                "end": 8638
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Routine_list",
                    "loc": {
                      "start": 8652,
                      "end": 8664
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8649,
                    "end": 8664
                  }
                }
              ],
              "loc": {
                "start": 8639,
                "end": 8670
              }
            },
            "loc": {
              "start": 8630,
              "end": 8670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContracts",
              "loc": {
                "start": 8675,
                "end": 8689
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "SmartContract_list",
                    "loc": {
                      "start": 8703,
                      "end": 8721
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8700,
                    "end": 8721
                  }
                }
              ],
              "loc": {
                "start": 8690,
                "end": 8727
              }
            },
            "loc": {
              "start": 8675,
              "end": 8727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standards",
              "loc": {
                "start": 8732,
                "end": 8741
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Standard_list",
                    "loc": {
                      "start": 8755,
                      "end": 8768
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8752,
                    "end": 8768
                  }
                }
              ],
              "loc": {
                "start": 8742,
                "end": 8774
              }
            },
            "loc": {
              "start": 8732,
              "end": 8774
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 8779,
                "end": 8784
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_list",
                    "loc": {
                      "start": 8798,
                      "end": 8807
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8795,
                    "end": 8807
                  }
                }
              ],
              "loc": {
                "start": 8785,
                "end": 8813
              }
            },
            "loc": {
              "start": 8779,
              "end": 8813
            }
          }
        ],
        "loc": {
          "start": 8401,
          "end": 8817
        }
      },
      "loc": {
        "start": 8378,
        "end": 8817
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 28,
          "end": 36
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 43,
                "end": 55
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 66,
                      "end": 68
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 66,
                    "end": 68
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 77,
                      "end": 85
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 77,
                    "end": 85
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 94,
                      "end": 101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 94,
                    "end": 101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 110,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 110,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 123,
                      "end": 130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 56,
                "end": 136
              }
            },
            "loc": {
              "start": 43,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 141,
                "end": 143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 141,
              "end": 143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 148,
                "end": 158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 148,
              "end": 158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 163,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 163,
              "end": 173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 178,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 178,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 191,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 209,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 209,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 231,
                "end": 241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 246,
                "end": 254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 259,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 259,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 273,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 273,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 290,
                "end": 302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 290,
              "end": 302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 307,
                "end": 319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 307,
              "end": 319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 338,
                      "end": 348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 338,
                    "end": 348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 357,
                      "end": 364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 357,
                    "end": 364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 373,
                      "end": 382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 391,
                      "end": 400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 409,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 409,
                    "end": 418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 427,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 442,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 449
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 455
              }
            },
            "loc": {
              "start": 324,
              "end": 455
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 457
        }
      },
      "loc": {
        "start": 28,
        "end": 457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 458,
          "end": 460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 458,
        "end": 460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 461,
          "end": 471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 461,
        "end": 471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 472,
          "end": 482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 472,
        "end": 482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 483,
          "end": 492
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 483,
        "end": 492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 493,
          "end": 504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 493,
        "end": 504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 505,
          "end": 511
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 521,
                "end": 531
              }
            },
            "directives": [],
            "loc": {
              "start": 518,
              "end": 531
            }
          }
        ],
        "loc": {
          "start": 512,
          "end": 533
        }
      },
      "loc": {
        "start": 505,
        "end": 533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 534,
          "end": 539
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 553,
                  "end": 565
                }
              },
              "loc": {
                "start": 553,
                "end": 565
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 579,
                      "end": 595
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 576,
                    "end": 595
                  }
                }
              ],
              "loc": {
                "start": 566,
                "end": 601
              }
            },
            "loc": {
              "start": 546,
              "end": 601
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 613,
                  "end": 617
                }
              },
              "loc": {
                "start": 613,
                "end": 617
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 631,
                      "end": 639
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 628,
                    "end": 639
                  }
                }
              ],
              "loc": {
                "start": 618,
                "end": 645
              }
            },
            "loc": {
              "start": 606,
              "end": 645
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 647
        }
      },
      "loc": {
        "start": 534,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 660,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 660,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 675,
          "end": 680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 675,
        "end": 680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 681,
          "end": 690
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 681,
        "end": 690
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 691,
          "end": 695
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 705,
                "end": 713
              }
            },
            "directives": [],
            "loc": {
              "start": 702,
              "end": 713
            }
          }
        ],
        "loc": {
          "start": 696,
          "end": 715
        }
      },
      "loc": {
        "start": 691,
        "end": 715
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 716,
          "end": 730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 716,
        "end": 730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 731,
          "end": 736
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 731,
        "end": 736
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 737,
          "end": 740
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 747,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 747,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 761,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 777,
                "end": 788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 793,
                "end": 802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 793,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 807,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 807,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 819,
                "end": 827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 819,
              "end": 827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 832,
                "end": 844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 832,
              "end": 844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 849,
                "end": 857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 849,
              "end": 857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 862,
                "end": 870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 862,
              "end": 870
            }
          }
        ],
        "loc": {
          "start": 741,
          "end": 872
        }
      },
      "loc": {
        "start": 737,
        "end": 872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 901,
          "end": 903
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 901,
        "end": 903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 904,
          "end": 913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 904,
        "end": 913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "apisCount",
        "loc": {
          "start": 947,
          "end": 956
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 947,
        "end": 956
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModesCount",
        "loc": {
          "start": 957,
          "end": 972
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 957,
        "end": 972
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 973,
          "end": 984
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 973,
        "end": 984
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetingsCount",
        "loc": {
          "start": 985,
          "end": 998
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 985,
        "end": 998
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "notesCount",
        "loc": {
          "start": 999,
          "end": 1009
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 999,
        "end": 1009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectsCount",
        "loc": {
          "start": 1010,
          "end": 1023
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1010,
        "end": 1023
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "routinesCount",
        "loc": {
          "start": 1024,
          "end": 1037
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1024,
        "end": 1037
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "schedulesCount",
        "loc": {
          "start": 1038,
          "end": 1052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1038,
        "end": 1052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "smartContractsCount",
        "loc": {
          "start": 1053,
          "end": 1072
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1053,
        "end": 1072
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "standardsCount",
        "loc": {
          "start": 1073,
          "end": 1087
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1073,
        "end": 1087
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1088,
          "end": 1090
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1088,
        "end": 1090
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1091,
          "end": 1101
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1091,
        "end": 1101
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1102,
          "end": 1112
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1102,
        "end": 1112
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1113,
          "end": 1118
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1113,
        "end": 1118
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1119,
          "end": 1124
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1119,
        "end": 1124
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1125,
          "end": 1130
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1144,
                  "end": 1156
                }
              },
              "loc": {
                "start": 1144,
                "end": 1156
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1170,
                      "end": 1186
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1167,
                    "end": 1186
                  }
                }
              ],
              "loc": {
                "start": 1157,
                "end": 1192
              }
            },
            "loc": {
              "start": 1137,
              "end": 1192
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1204,
                  "end": 1208
                }
              },
              "loc": {
                "start": 1204,
                "end": 1208
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1222,
                      "end": 1230
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1219,
                    "end": 1230
                  }
                }
              ],
              "loc": {
                "start": 1209,
                "end": 1236
              }
            },
            "loc": {
              "start": 1197,
              "end": 1236
            }
          }
        ],
        "loc": {
          "start": 1131,
          "end": 1238
        }
      },
      "loc": {
        "start": 1125,
        "end": 1238
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1239,
          "end": 1242
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1249,
                "end": 1258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1249,
              "end": 1258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1263,
                "end": 1272
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1263,
              "end": 1272
            }
          }
        ],
        "loc": {
          "start": 1243,
          "end": 1274
        }
      },
      "loc": {
        "start": 1239,
        "end": 1274
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1308,
          "end": 1310
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1308,
        "end": 1310
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1311,
          "end": 1321
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1311,
        "end": 1321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1322,
          "end": 1332
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1322,
        "end": 1332
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1333,
          "end": 1338
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1333,
        "end": 1338
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1339,
          "end": 1344
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1339,
        "end": 1344
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1345,
          "end": 1350
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1364,
                  "end": 1376
                }
              },
              "loc": {
                "start": 1364,
                "end": 1376
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1390,
                      "end": 1406
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1387,
                    "end": 1406
                  }
                }
              ],
              "loc": {
                "start": 1377,
                "end": 1412
              }
            },
            "loc": {
              "start": 1357,
              "end": 1412
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1424,
                  "end": 1428
                }
              },
              "loc": {
                "start": 1424,
                "end": 1428
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1442,
                      "end": 1450
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1439,
                    "end": 1450
                  }
                }
              ],
              "loc": {
                "start": 1429,
                "end": 1456
              }
            },
            "loc": {
              "start": 1417,
              "end": 1456
            }
          }
        ],
        "loc": {
          "start": 1351,
          "end": 1458
        }
      },
      "loc": {
        "start": 1345,
        "end": 1458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1459,
          "end": 1462
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1469,
                "end": 1478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1469,
              "end": 1478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1483,
                "end": 1492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1483,
              "end": 1492
            }
          }
        ],
        "loc": {
          "start": 1463,
          "end": 1494
        }
      },
      "loc": {
        "start": 1459,
        "end": 1494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1526,
          "end": 1534
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1541,
                "end": 1553
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1564,
                      "end": 1566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1564,
                    "end": 1566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1575,
                      "end": 1583
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1575,
                    "end": 1583
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1592,
                      "end": 1603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1592,
                    "end": 1603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1612,
                      "end": 1616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1612,
                    "end": 1616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "text",
                    "loc": {
                      "start": 1625,
                      "end": 1629
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1625,
                    "end": 1629
                  }
                }
              ],
              "loc": {
                "start": 1554,
                "end": 1635
              }
            },
            "loc": {
              "start": 1541,
              "end": 1635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1640,
                "end": 1642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1640,
              "end": 1642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1647,
                "end": 1657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1647,
              "end": 1657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1662,
                "end": 1672
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1662,
              "end": 1672
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1677,
                "end": 1685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1677,
              "end": 1685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1690,
                "end": 1699
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1690,
              "end": 1699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1704,
                "end": 1716
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1704,
              "end": 1716
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1721,
                "end": 1733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1721,
              "end": 1733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1738,
                "end": 1750
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1738,
              "end": 1750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1755,
                "end": 1758
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 1769,
                      "end": 1779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1769,
                    "end": 1779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1788,
                      "end": 1795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1788,
                    "end": 1795
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1804,
                      "end": 1813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1804,
                    "end": 1813
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1822,
                      "end": 1831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1822,
                    "end": 1831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1840,
                      "end": 1849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1840,
                    "end": 1849
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1858,
                      "end": 1864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1858,
                    "end": 1864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1873,
                      "end": 1880
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1873,
                    "end": 1880
                  }
                }
              ],
              "loc": {
                "start": 1759,
                "end": 1886
              }
            },
            "loc": {
              "start": 1755,
              "end": 1886
            }
          }
        ],
        "loc": {
          "start": 1535,
          "end": 1888
        }
      },
      "loc": {
        "start": 1526,
        "end": 1888
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1889,
          "end": 1891
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1889,
        "end": 1891
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1892,
          "end": 1902
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1892,
        "end": 1902
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1903,
          "end": 1913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1903,
        "end": 1913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1914,
          "end": 1923
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1914,
        "end": 1923
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1924,
          "end": 1935
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1924,
        "end": 1935
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1936,
          "end": 1942
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 1952,
                "end": 1962
              }
            },
            "directives": [],
            "loc": {
              "start": 1949,
              "end": 1962
            }
          }
        ],
        "loc": {
          "start": 1943,
          "end": 1964
        }
      },
      "loc": {
        "start": 1936,
        "end": 1964
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1965,
          "end": 1970
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1984,
                  "end": 1996
                }
              },
              "loc": {
                "start": 1984,
                "end": 1996
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 2010,
                      "end": 2026
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2007,
                    "end": 2026
                  }
                }
              ],
              "loc": {
                "start": 1997,
                "end": 2032
              }
            },
            "loc": {
              "start": 1977,
              "end": 2032
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 2044,
                  "end": 2048
                }
              },
              "loc": {
                "start": 2044,
                "end": 2048
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 2062,
                      "end": 2070
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2059,
                    "end": 2070
                  }
                }
              ],
              "loc": {
                "start": 2049,
                "end": 2076
              }
            },
            "loc": {
              "start": 2037,
              "end": 2076
            }
          }
        ],
        "loc": {
          "start": 1971,
          "end": 2078
        }
      },
      "loc": {
        "start": 1965,
        "end": 2078
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2079,
          "end": 2090
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2079,
        "end": 2090
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2091,
          "end": 2105
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2091,
        "end": 2105
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2106,
          "end": 2111
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2106,
        "end": 2111
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2112,
          "end": 2121
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2112,
        "end": 2121
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2122,
          "end": 2126
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 2136,
                "end": 2144
              }
            },
            "directives": [],
            "loc": {
              "start": 2133,
              "end": 2144
            }
          }
        ],
        "loc": {
          "start": 2127,
          "end": 2146
        }
      },
      "loc": {
        "start": 2122,
        "end": 2146
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2147,
          "end": 2161
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2147,
        "end": 2161
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2162,
          "end": 2167
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2162,
        "end": 2167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2168,
          "end": 2171
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2178,
                "end": 2187
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2178,
              "end": 2187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2192,
                "end": 2203
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2192,
              "end": 2203
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 2208,
                "end": 2219
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2208,
              "end": 2219
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2224,
                "end": 2233
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2224,
              "end": 2233
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2238,
                "end": 2245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2238,
              "end": 2245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2250,
                "end": 2258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2250,
              "end": 2258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2263,
                "end": 2275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2263,
              "end": 2275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2280,
                "end": 2288
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2280,
              "end": 2288
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2293,
                "end": 2301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2293,
              "end": 2301
            }
          }
        ],
        "loc": {
          "start": 2172,
          "end": 2303
        }
      },
      "loc": {
        "start": 2168,
        "end": 2303
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2334,
          "end": 2336
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2334,
        "end": 2336
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2337,
          "end": 2346
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2337,
        "end": 2346
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2394,
          "end": 2396
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2394,
        "end": 2396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2397,
          "end": 2403
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2397,
        "end": 2403
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2404,
          "end": 2414
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2404,
        "end": 2414
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2415,
          "end": 2425
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2415,
        "end": 2425
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 2426,
          "end": 2444
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2426,
        "end": 2444
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2445,
          "end": 2454
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2445,
        "end": 2454
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 2455,
          "end": 2468
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2455,
        "end": 2468
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 2469,
          "end": 2481
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2469,
        "end": 2481
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 2482,
          "end": 2494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2482,
        "end": 2494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2495,
          "end": 2504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2495,
        "end": 2504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2505,
          "end": 2509
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 2519,
                "end": 2527
              }
            },
            "directives": [],
            "loc": {
              "start": 2516,
              "end": 2527
            }
          }
        ],
        "loc": {
          "start": 2510,
          "end": 2529
        }
      },
      "loc": {
        "start": 2505,
        "end": 2529
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2530,
          "end": 2542
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2549,
                "end": 2551
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2549,
              "end": 2551
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2556,
                "end": 2564
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2556,
              "end": 2564
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 2569,
                "end": 2572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2569,
              "end": 2572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2577,
                "end": 2581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2577,
              "end": 2581
            }
          }
        ],
        "loc": {
          "start": 2543,
          "end": 2583
        }
      },
      "loc": {
        "start": 2530,
        "end": 2583
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2584,
          "end": 2587
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 2594,
                "end": 2607
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2594,
              "end": 2607
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2612,
                "end": 2621
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2612,
              "end": 2621
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2626,
                "end": 2637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2626,
              "end": 2637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2642,
                "end": 2651
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2642,
              "end": 2651
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2656,
                "end": 2665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2656,
              "end": 2665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2670,
                "end": 2677
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2670,
              "end": 2677
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2682,
                "end": 2694
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2682,
              "end": 2694
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2699,
                "end": 2707
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2699,
              "end": 2707
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2712,
                "end": 2726
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2737,
                      "end": 2739
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2737,
                    "end": 2739
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2748,
                      "end": 2758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2748,
                    "end": 2758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2767,
                      "end": 2777
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2767,
                    "end": 2777
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2786,
                      "end": 2793
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2786,
                    "end": 2793
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2802,
                      "end": 2813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2802,
                    "end": 2813
                  }
                }
              ],
              "loc": {
                "start": 2727,
                "end": 2819
              }
            },
            "loc": {
              "start": 2712,
              "end": 2819
            }
          }
        ],
        "loc": {
          "start": 2588,
          "end": 2821
        }
      },
      "loc": {
        "start": 2584,
        "end": 2821
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2868,
          "end": 2870
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2868,
        "end": 2870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2871,
          "end": 2877
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2871,
        "end": 2877
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2878,
          "end": 2881
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 2888,
                "end": 2901
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2888,
              "end": 2901
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2906,
                "end": 2915
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2906,
              "end": 2915
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2920,
                "end": 2931
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2920,
              "end": 2931
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2936,
                "end": 2945
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2936,
              "end": 2945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2950,
                "end": 2959
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2950,
              "end": 2959
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2964,
                "end": 2971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2964,
              "end": 2971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2976,
                "end": 2988
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2976,
              "end": 2988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2993,
                "end": 3001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2993,
              "end": 3001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 3006,
                "end": 3020
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3031,
                      "end": 3033
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3031,
                    "end": 3033
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3042,
                      "end": 3052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3042,
                    "end": 3052
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3061,
                      "end": 3071
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3061,
                    "end": 3071
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 3080,
                      "end": 3087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3080,
                    "end": 3087
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 3096,
                      "end": 3107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3096,
                    "end": 3107
                  }
                }
              ],
              "loc": {
                "start": 3021,
                "end": 3113
              }
            },
            "loc": {
              "start": 3006,
              "end": 3113
            }
          }
        ],
        "loc": {
          "start": 2882,
          "end": 3115
        }
      },
      "loc": {
        "start": 2878,
        "end": 3115
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 3153,
          "end": 3161
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3168,
                "end": 3180
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3191,
                      "end": 3193
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3191,
                    "end": 3193
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3202,
                      "end": 3210
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3202,
                    "end": 3210
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3219,
                      "end": 3230
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3219,
                    "end": 3230
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3239,
                      "end": 3243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3239,
                    "end": 3243
                  }
                }
              ],
              "loc": {
                "start": 3181,
                "end": 3249
              }
            },
            "loc": {
              "start": 3168,
              "end": 3249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3254,
                "end": 3256
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3254,
              "end": 3256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3261,
                "end": 3271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3261,
              "end": 3271
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3276,
                "end": 3286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3276,
              "end": 3286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3291,
                "end": 3307
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3291,
              "end": 3307
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3312,
                "end": 3320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3312,
              "end": 3320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3325,
                "end": 3334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3325,
              "end": 3334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3339,
                "end": 3351
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3339,
              "end": 3351
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3356,
                "end": 3372
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3356,
              "end": 3372
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3377,
                "end": 3387
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3377,
              "end": 3387
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3392,
                "end": 3404
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3392,
              "end": 3404
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3409,
                "end": 3421
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3409,
              "end": 3421
            }
          }
        ],
        "loc": {
          "start": 3162,
          "end": 3423
        }
      },
      "loc": {
        "start": 3153,
        "end": 3423
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3424,
          "end": 3426
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3424,
        "end": 3426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3427,
          "end": 3437
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3427,
        "end": 3437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3438,
          "end": 3448
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3438,
        "end": 3448
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3449,
          "end": 3458
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3449,
        "end": 3458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3459,
          "end": 3470
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3459,
        "end": 3470
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3471,
          "end": 3477
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 3487,
                "end": 3497
              }
            },
            "directives": [],
            "loc": {
              "start": 3484,
              "end": 3497
            }
          }
        ],
        "loc": {
          "start": 3478,
          "end": 3499
        }
      },
      "loc": {
        "start": 3471,
        "end": 3499
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3500,
          "end": 3505
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 3519,
                  "end": 3531
                }
              },
              "loc": {
                "start": 3519,
                "end": 3531
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 3545,
                      "end": 3561
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3542,
                    "end": 3561
                  }
                }
              ],
              "loc": {
                "start": 3532,
                "end": 3567
              }
            },
            "loc": {
              "start": 3512,
              "end": 3567
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 3579,
                  "end": 3583
                }
              },
              "loc": {
                "start": 3579,
                "end": 3583
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 3597,
                      "end": 3605
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3594,
                    "end": 3605
                  }
                }
              ],
              "loc": {
                "start": 3584,
                "end": 3611
              }
            },
            "loc": {
              "start": 3572,
              "end": 3611
            }
          }
        ],
        "loc": {
          "start": 3506,
          "end": 3613
        }
      },
      "loc": {
        "start": 3500,
        "end": 3613
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3614,
          "end": 3625
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3614,
        "end": 3625
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3626,
          "end": 3640
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3626,
        "end": 3640
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3641,
          "end": 3646
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3641,
        "end": 3646
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3647,
          "end": 3656
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3647,
        "end": 3656
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3657,
          "end": 3661
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 3671,
                "end": 3679
              }
            },
            "directives": [],
            "loc": {
              "start": 3668,
              "end": 3679
            }
          }
        ],
        "loc": {
          "start": 3662,
          "end": 3681
        }
      },
      "loc": {
        "start": 3657,
        "end": 3681
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3682,
          "end": 3696
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3682,
        "end": 3696
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3697,
          "end": 3702
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3697,
        "end": 3702
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3703,
          "end": 3706
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 3713,
                "end": 3722
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3713,
              "end": 3722
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3727,
                "end": 3738
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3727,
              "end": 3738
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3743,
                "end": 3754
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3743,
              "end": 3754
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3759,
                "end": 3768
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3759,
              "end": 3768
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3773,
                "end": 3780
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3773,
              "end": 3780
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3785,
                "end": 3793
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3785,
              "end": 3793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3798,
                "end": 3810
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3798,
              "end": 3810
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3815,
                "end": 3823
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3815,
              "end": 3823
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3828,
                "end": 3836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3828,
              "end": 3836
            }
          }
        ],
        "loc": {
          "start": 3707,
          "end": 3838
        }
      },
      "loc": {
        "start": 3703,
        "end": 3838
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3875,
          "end": 3877
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3875,
        "end": 3877
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3878,
          "end": 3887
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3878,
        "end": 3887
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3927,
          "end": 3939
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3946,
                "end": 3948
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3946,
              "end": 3948
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3953,
                "end": 3961
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3953,
              "end": 3961
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3966,
                "end": 3977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3966,
              "end": 3977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3982,
                "end": 3986
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3982,
              "end": 3986
            }
          }
        ],
        "loc": {
          "start": 3940,
          "end": 3988
        }
      },
      "loc": {
        "start": 3927,
        "end": 3988
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3989,
          "end": 3991
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3989,
        "end": 3991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3992,
          "end": 4002
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3992,
        "end": 4002
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4003,
          "end": 4013
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4003,
        "end": 4013
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 4014,
          "end": 4023
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4030,
                "end": 4032
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4030,
              "end": 4032
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 4037,
                "end": 4042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4037,
              "end": 4042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 4047,
                "end": 4051
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4047,
              "end": 4051
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 4056,
                "end": 4062
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4056,
              "end": 4062
            }
          }
        ],
        "loc": {
          "start": 4024,
          "end": 4064
        }
      },
      "loc": {
        "start": 4014,
        "end": 4064
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 4065,
          "end": 4082
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4065,
        "end": 4082
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4083,
          "end": 4092
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4083,
        "end": 4092
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4093,
          "end": 4098
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4093,
        "end": 4098
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4099,
          "end": 4108
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4099,
        "end": 4108
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 4109,
          "end": 4121
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4109,
        "end": 4121
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 4122,
          "end": 4135
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4122,
        "end": 4135
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 4136,
          "end": 4148
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4136,
        "end": 4148
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 4149,
          "end": 4158
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Api",
                "loc": {
                  "start": 4172,
                  "end": 4175
                }
              },
              "loc": {
                "start": 4172,
                "end": 4175
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Api_nav",
                    "loc": {
                      "start": 4189,
                      "end": 4196
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4186,
                    "end": 4196
                  }
                }
              ],
              "loc": {
                "start": 4176,
                "end": 4202
              }
            },
            "loc": {
              "start": 4165,
              "end": 4202
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Note",
                "loc": {
                  "start": 4214,
                  "end": 4218
                }
              },
              "loc": {
                "start": 4214,
                "end": 4218
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_nav",
                    "loc": {
                      "start": 4232,
                      "end": 4240
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4229,
                    "end": 4240
                  }
                }
              ],
              "loc": {
                "start": 4219,
                "end": 4246
              }
            },
            "loc": {
              "start": 4207,
              "end": 4246
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 4258,
                  "end": 4270
                }
              },
              "loc": {
                "start": 4258,
                "end": 4270
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 4284,
                      "end": 4300
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4281,
                    "end": 4300
                  }
                }
              ],
              "loc": {
                "start": 4271,
                "end": 4306
              }
            },
            "loc": {
              "start": 4251,
              "end": 4306
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Project",
                "loc": {
                  "start": 4318,
                  "end": 4325
                }
              },
              "loc": {
                "start": 4318,
                "end": 4325
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Project_nav",
                    "loc": {
                      "start": 4339,
                      "end": 4350
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4336,
                    "end": 4350
                  }
                }
              ],
              "loc": {
                "start": 4326,
                "end": 4356
              }
            },
            "loc": {
              "start": 4311,
              "end": 4356
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Routine",
                "loc": {
                  "start": 4368,
                  "end": 4375
                }
              },
              "loc": {
                "start": 4368,
                "end": 4375
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Routine_nav",
                    "loc": {
                      "start": 4389,
                      "end": 4400
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4386,
                    "end": 4400
                  }
                }
              ],
              "loc": {
                "start": 4376,
                "end": 4406
              }
            },
            "loc": {
              "start": 4361,
              "end": 4406
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "SmartContract",
                "loc": {
                  "start": 4418,
                  "end": 4431
                }
              },
              "loc": {
                "start": 4418,
                "end": 4431
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "SmartContract_nav",
                    "loc": {
                      "start": 4445,
                      "end": 4462
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4442,
                    "end": 4462
                  }
                }
              ],
              "loc": {
                "start": 4432,
                "end": 4468
              }
            },
            "loc": {
              "start": 4411,
              "end": 4468
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Standard",
                "loc": {
                  "start": 4480,
                  "end": 4488
                }
              },
              "loc": {
                "start": 4480,
                "end": 4488
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Standard_nav",
                    "loc": {
                      "start": 4502,
                      "end": 4514
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4499,
                    "end": 4514
                  }
                }
              ],
              "loc": {
                "start": 4489,
                "end": 4520
              }
            },
            "loc": {
              "start": 4473,
              "end": 4520
            }
          }
        ],
        "loc": {
          "start": 4159,
          "end": 4522
        }
      },
      "loc": {
        "start": 4149,
        "end": 4522
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4523,
          "end": 4527
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 4537,
                "end": 4545
              }
            },
            "directives": [],
            "loc": {
              "start": 4534,
              "end": 4545
            }
          }
        ],
        "loc": {
          "start": 4528,
          "end": 4547
        }
      },
      "loc": {
        "start": 4523,
        "end": 4547
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4548,
          "end": 4551
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 4558,
                "end": 4566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4558,
              "end": 4566
            }
          }
        ],
        "loc": {
          "start": 4552,
          "end": 4568
        }
      },
      "loc": {
        "start": 4548,
        "end": 4568
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4606,
          "end": 4614
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 4621,
                "end": 4633
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4644,
                      "end": 4646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4644,
                    "end": 4646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4655,
                      "end": 4663
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4655,
                    "end": 4663
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4672,
                      "end": 4683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4672,
                    "end": 4683
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4692,
                      "end": 4704
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4692,
                    "end": 4704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4713,
                      "end": 4717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4713,
                    "end": 4717
                  }
                }
              ],
              "loc": {
                "start": 4634,
                "end": 4723
              }
            },
            "loc": {
              "start": 4621,
              "end": 4723
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4728,
                "end": 4730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4728,
              "end": 4730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4735,
                "end": 4745
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4735,
              "end": 4745
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4750,
                "end": 4760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4750,
              "end": 4760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4765,
                "end": 4776
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4765,
              "end": 4776
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4781,
                "end": 4794
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4781,
              "end": 4794
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4799,
                "end": 4809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4799,
              "end": 4809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4814,
                "end": 4823
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4814,
              "end": 4823
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4828,
                "end": 4836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4828,
              "end": 4836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4841,
                "end": 4850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4841,
              "end": 4850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4855,
                "end": 4865
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4855,
              "end": 4865
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4870,
                "end": 4882
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4870,
              "end": 4882
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4887,
                "end": 4901
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4887,
              "end": 4901
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 4906,
                "end": 4927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4906,
              "end": 4927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 4932,
                "end": 4943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4932,
              "end": 4943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4948,
                "end": 4960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4948,
              "end": 4960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4965,
                "end": 4977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4965,
              "end": 4977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4982,
                "end": 4995
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4982,
              "end": 4995
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5000,
                "end": 5022
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5000,
              "end": 5022
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5027,
                "end": 5037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5027,
              "end": 5037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 5042,
                "end": 5053
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5042,
              "end": 5053
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 5058,
                "end": 5068
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5058,
              "end": 5068
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 5073,
                "end": 5087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5073,
              "end": 5087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 5092,
                "end": 5104
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5092,
              "end": 5104
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5109,
                "end": 5121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5109,
              "end": 5121
            }
          }
        ],
        "loc": {
          "start": 4615,
          "end": 5123
        }
      },
      "loc": {
        "start": 4606,
        "end": 5123
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5124,
          "end": 5126
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5124,
        "end": 5126
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5127,
          "end": 5137
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5127,
        "end": 5137
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5138,
          "end": 5148
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5138,
        "end": 5148
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5149,
          "end": 5159
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5149,
        "end": 5159
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5160,
          "end": 5169
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5160,
        "end": 5169
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5170,
          "end": 5181
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5170,
        "end": 5181
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5182,
          "end": 5188
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 5198,
                "end": 5208
              }
            },
            "directives": [],
            "loc": {
              "start": 5195,
              "end": 5208
            }
          }
        ],
        "loc": {
          "start": 5189,
          "end": 5210
        }
      },
      "loc": {
        "start": 5182,
        "end": 5210
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5211,
          "end": 5216
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 5230,
                  "end": 5242
                }
              },
              "loc": {
                "start": 5230,
                "end": 5242
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 5256,
                      "end": 5272
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5253,
                    "end": 5272
                  }
                }
              ],
              "loc": {
                "start": 5243,
                "end": 5278
              }
            },
            "loc": {
              "start": 5223,
              "end": 5278
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 5290,
                  "end": 5294
                }
              },
              "loc": {
                "start": 5290,
                "end": 5294
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 5308,
                      "end": 5316
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5305,
                    "end": 5316
                  }
                }
              ],
              "loc": {
                "start": 5295,
                "end": 5322
              }
            },
            "loc": {
              "start": 5283,
              "end": 5322
            }
          }
        ],
        "loc": {
          "start": 5217,
          "end": 5324
        }
      },
      "loc": {
        "start": 5211,
        "end": 5324
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5325,
          "end": 5336
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5325,
        "end": 5336
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5337,
          "end": 5351
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5337,
        "end": 5351
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5352,
          "end": 5357
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5352,
        "end": 5357
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5358,
          "end": 5367
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5358,
        "end": 5367
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5368,
          "end": 5372
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 5382,
                "end": 5390
              }
            },
            "directives": [],
            "loc": {
              "start": 5379,
              "end": 5390
            }
          }
        ],
        "loc": {
          "start": 5373,
          "end": 5392
        }
      },
      "loc": {
        "start": 5368,
        "end": 5392
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5393,
          "end": 5407
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5393,
        "end": 5407
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5408,
          "end": 5413
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5408,
        "end": 5413
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5414,
          "end": 5417
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canComment",
              "loc": {
                "start": 5424,
                "end": 5434
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5424,
              "end": 5434
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5439,
                "end": 5448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5439,
              "end": 5448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5453,
                "end": 5464
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5453,
              "end": 5464
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5469,
                "end": 5478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5469,
              "end": 5478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5483,
                "end": 5490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5483,
              "end": 5490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5495,
                "end": 5503
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5495,
              "end": 5503
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5508,
                "end": 5520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5508,
              "end": 5520
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5525,
                "end": 5533
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5525,
              "end": 5533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5538,
                "end": 5546
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5538,
              "end": 5546
            }
          }
        ],
        "loc": {
          "start": 5418,
          "end": 5548
        }
      },
      "loc": {
        "start": 5414,
        "end": 5548
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5585,
          "end": 5587
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5585,
        "end": 5587
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5588,
          "end": 5598
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5588,
        "end": 5598
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5599,
          "end": 5608
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5599,
        "end": 5608
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5650,
          "end": 5652
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5650,
        "end": 5652
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5653,
          "end": 5663
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5653,
        "end": 5663
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5664,
          "end": 5674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5664,
        "end": 5674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 5675,
          "end": 5684
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5675,
        "end": 5684
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 5685,
          "end": 5692
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5685,
        "end": 5692
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 5693,
          "end": 5701
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5693,
        "end": 5701
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 5702,
          "end": 5712
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5719,
                "end": 5721
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5719,
              "end": 5721
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 5726,
                "end": 5743
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5726,
              "end": 5743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 5748,
                "end": 5760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5748,
              "end": 5760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 5765,
                "end": 5775
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5765,
              "end": 5775
            }
          }
        ],
        "loc": {
          "start": 5713,
          "end": 5777
        }
      },
      "loc": {
        "start": 5702,
        "end": 5777
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 5778,
          "end": 5789
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5796,
                "end": 5798
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5796,
              "end": 5798
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 5803,
                "end": 5817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5803,
              "end": 5817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 5822,
                "end": 5830
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5822,
              "end": 5830
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 5835,
                "end": 5844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5835,
              "end": 5844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 5849,
                "end": 5859
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5849,
              "end": 5859
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 5864,
                "end": 5869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5864,
              "end": 5869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 5874,
                "end": 5881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5874,
              "end": 5881
            }
          }
        ],
        "loc": {
          "start": 5790,
          "end": 5883
        }
      },
      "loc": {
        "start": 5778,
        "end": 5883
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5933,
          "end": 5941
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 5948,
                "end": 5960
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5971,
                      "end": 5973
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5971,
                    "end": 5973
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5982,
                      "end": 5990
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5982,
                    "end": 5990
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5999,
                      "end": 6010
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5999,
                    "end": 6010
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6019,
                      "end": 6031
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6019,
                    "end": 6031
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6040,
                      "end": 6044
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6040,
                    "end": 6044
                  }
                }
              ],
              "loc": {
                "start": 5961,
                "end": 6050
              }
            },
            "loc": {
              "start": 5948,
              "end": 6050
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6055,
                "end": 6057
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6055,
              "end": 6057
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6062,
                "end": 6072
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6062,
              "end": 6072
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6077,
                "end": 6087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6077,
              "end": 6087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 6092,
                "end": 6102
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6092,
              "end": 6102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 6107,
                "end": 6116
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6107,
              "end": 6116
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6121,
                "end": 6129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6121,
              "end": 6129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6134,
                "end": 6143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6134,
              "end": 6143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6148,
                "end": 6155
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6148,
              "end": 6155
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contractType",
              "loc": {
                "start": 6160,
                "end": 6172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6160,
              "end": 6172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "content",
              "loc": {
                "start": 6177,
                "end": 6184
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6177,
              "end": 6184
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6189,
                "end": 6201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6189,
              "end": 6201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6206,
                "end": 6218
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6206,
              "end": 6218
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6223,
                "end": 6236
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6223,
              "end": 6236
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6241,
                "end": 6263
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6241,
              "end": 6263
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6268,
                "end": 6278
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6268,
              "end": 6278
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6283,
                "end": 6295
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6283,
              "end": 6295
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6300,
                "end": 6303
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 6314,
                      "end": 6324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6314,
                    "end": 6324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6333,
                      "end": 6340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6333,
                    "end": 6340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6349,
                      "end": 6358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6349,
                    "end": 6358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6367,
                      "end": 6376
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6367,
                    "end": 6376
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6385,
                      "end": 6394
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6385,
                    "end": 6394
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6403,
                      "end": 6409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6403,
                    "end": 6409
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6418,
                      "end": 6425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6418,
                    "end": 6425
                  }
                }
              ],
              "loc": {
                "start": 6304,
                "end": 6431
              }
            },
            "loc": {
              "start": 6300,
              "end": 6431
            }
          }
        ],
        "loc": {
          "start": 5942,
          "end": 6433
        }
      },
      "loc": {
        "start": 5933,
        "end": 6433
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6434,
          "end": 6436
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6434,
        "end": 6436
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6437,
          "end": 6447
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6437,
        "end": 6447
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6448,
          "end": 6458
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6448,
        "end": 6458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6459,
          "end": 6468
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6459,
        "end": 6468
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6469,
          "end": 6480
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6469,
        "end": 6480
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6481,
          "end": 6487
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 6497,
                "end": 6507
              }
            },
            "directives": [],
            "loc": {
              "start": 6494,
              "end": 6507
            }
          }
        ],
        "loc": {
          "start": 6488,
          "end": 6509
        }
      },
      "loc": {
        "start": 6481,
        "end": 6509
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6510,
          "end": 6515
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 6529,
                  "end": 6541
                }
              },
              "loc": {
                "start": 6529,
                "end": 6541
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6555,
                      "end": 6571
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6552,
                    "end": 6571
                  }
                }
              ],
              "loc": {
                "start": 6542,
                "end": 6577
              }
            },
            "loc": {
              "start": 6522,
              "end": 6577
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 6589,
                  "end": 6593
                }
              },
              "loc": {
                "start": 6589,
                "end": 6593
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6607,
                      "end": 6615
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6604,
                    "end": 6615
                  }
                }
              ],
              "loc": {
                "start": 6594,
                "end": 6621
              }
            },
            "loc": {
              "start": 6582,
              "end": 6621
            }
          }
        ],
        "loc": {
          "start": 6516,
          "end": 6623
        }
      },
      "loc": {
        "start": 6510,
        "end": 6623
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6624,
          "end": 6635
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6624,
        "end": 6635
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6636,
          "end": 6650
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6636,
        "end": 6650
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6651,
          "end": 6656
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6651,
        "end": 6656
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6657,
          "end": 6666
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6657,
        "end": 6666
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6667,
          "end": 6671
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 6681,
                "end": 6689
              }
            },
            "directives": [],
            "loc": {
              "start": 6678,
              "end": 6689
            }
          }
        ],
        "loc": {
          "start": 6672,
          "end": 6691
        }
      },
      "loc": {
        "start": 6667,
        "end": 6691
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6692,
          "end": 6706
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6692,
        "end": 6706
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6707,
          "end": 6712
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6707,
        "end": 6712
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6713,
          "end": 6716
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 6723,
                "end": 6732
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6723,
              "end": 6732
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6737,
                "end": 6748
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6737,
              "end": 6748
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6753,
                "end": 6764
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6753,
              "end": 6764
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6769,
                "end": 6778
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6769,
              "end": 6778
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6783,
                "end": 6790
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6783,
              "end": 6790
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6795,
                "end": 6803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6795,
              "end": 6803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6808,
                "end": 6820
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6808,
              "end": 6820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6825,
                "end": 6833
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6825,
              "end": 6833
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6838,
                "end": 6846
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6838,
              "end": 6846
            }
          }
        ],
        "loc": {
          "start": 6717,
          "end": 6848
        }
      },
      "loc": {
        "start": 6713,
        "end": 6848
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6897,
          "end": 6899
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6897,
        "end": 6899
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6900,
          "end": 6909
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6900,
        "end": 6909
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 6949,
          "end": 6957
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6964,
                "end": 6976
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6987,
                      "end": 6989
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6987,
                    "end": 6989
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6998,
                      "end": 7006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6998,
                    "end": 7006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 7015,
                      "end": 7026
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7015,
                    "end": 7026
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 7035,
                      "end": 7047
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7035,
                    "end": 7047
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7056,
                      "end": 7060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7056,
                    "end": 7060
                  }
                }
              ],
              "loc": {
                "start": 6977,
                "end": 7066
              }
            },
            "loc": {
              "start": 6964,
              "end": 7066
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7071,
                "end": 7073
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7071,
              "end": 7073
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7078,
                "end": 7088
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7078,
              "end": 7088
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7093,
                "end": 7103
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7093,
              "end": 7103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 7108,
                "end": 7118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7108,
              "end": 7118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 7123,
                "end": 7129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7123,
              "end": 7129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 7134,
                "end": 7142
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7134,
              "end": 7142
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7147,
                "end": 7156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7147,
              "end": 7156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 7161,
                "end": 7168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7161,
              "end": 7168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 7173,
                "end": 7185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7173,
              "end": 7185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 7190,
                "end": 7195
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7190,
              "end": 7195
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 7200,
                "end": 7203
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7200,
              "end": 7203
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 7208,
                "end": 7220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7208,
              "end": 7220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 7225,
                "end": 7237
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7225,
              "end": 7237
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 7242,
                "end": 7255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7242,
              "end": 7255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 7260,
                "end": 7282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7260,
              "end": 7282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 7287,
                "end": 7297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7287,
              "end": 7297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 7302,
                "end": 7314
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7302,
              "end": 7314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7319,
                "end": 7322
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 7333,
                      "end": 7343
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7333,
                    "end": 7343
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 7352,
                      "end": 7359
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7352,
                    "end": 7359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7368,
                      "end": 7377
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7368,
                    "end": 7377
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7386,
                      "end": 7395
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7386,
                    "end": 7395
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7404,
                      "end": 7413
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7404,
                    "end": 7413
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 7422,
                      "end": 7428
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7422,
                    "end": 7428
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7437,
                      "end": 7444
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7437,
                    "end": 7444
                  }
                }
              ],
              "loc": {
                "start": 7323,
                "end": 7450
              }
            },
            "loc": {
              "start": 7319,
              "end": 7450
            }
          }
        ],
        "loc": {
          "start": 6958,
          "end": 7452
        }
      },
      "loc": {
        "start": 6949,
        "end": 7452
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7453,
          "end": 7455
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7453,
        "end": 7455
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7456,
          "end": 7466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7456,
        "end": 7466
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7467,
          "end": 7477
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7467,
        "end": 7477
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7478,
          "end": 7487
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7478,
        "end": 7487
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 7488,
          "end": 7499
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7488,
        "end": 7499
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 7500,
          "end": 7506
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 7516,
                "end": 7526
              }
            },
            "directives": [],
            "loc": {
              "start": 7513,
              "end": 7526
            }
          }
        ],
        "loc": {
          "start": 7507,
          "end": 7528
        }
      },
      "loc": {
        "start": 7500,
        "end": 7528
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 7529,
          "end": 7534
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 7548,
                  "end": 7560
                }
              },
              "loc": {
                "start": 7548,
                "end": 7560
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7574,
                      "end": 7590
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7571,
                    "end": 7590
                  }
                }
              ],
              "loc": {
                "start": 7561,
                "end": 7596
              }
            },
            "loc": {
              "start": 7541,
              "end": 7596
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 7608,
                  "end": 7612
                }
              },
              "loc": {
                "start": 7608,
                "end": 7612
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 7626,
                      "end": 7634
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7623,
                    "end": 7634
                  }
                }
              ],
              "loc": {
                "start": 7613,
                "end": 7640
              }
            },
            "loc": {
              "start": 7601,
              "end": 7640
            }
          }
        ],
        "loc": {
          "start": 7535,
          "end": 7642
        }
      },
      "loc": {
        "start": 7529,
        "end": 7642
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 7643,
          "end": 7654
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7643,
        "end": 7654
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 7655,
          "end": 7669
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7655,
        "end": 7669
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 7670,
          "end": 7675
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7670,
        "end": 7675
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7676,
          "end": 7685
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7676,
        "end": 7685
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 7686,
          "end": 7690
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 7700,
                "end": 7708
              }
            },
            "directives": [],
            "loc": {
              "start": 7697,
              "end": 7708
            }
          }
        ],
        "loc": {
          "start": 7691,
          "end": 7710
        }
      },
      "loc": {
        "start": 7686,
        "end": 7710
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 7711,
          "end": 7725
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7711,
        "end": 7725
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 7726,
          "end": 7731
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7726,
        "end": 7731
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7732,
          "end": 7735
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 7742,
                "end": 7751
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7742,
              "end": 7751
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7756,
                "end": 7767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7756,
              "end": 7767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 7772,
                "end": 7783
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7772,
              "end": 7783
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7788,
                "end": 7797
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7788,
              "end": 7797
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7802,
                "end": 7809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7802,
              "end": 7809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 7814,
                "end": 7822
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7814,
              "end": 7822
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7827,
                "end": 7839
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7827,
              "end": 7839
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7844,
                "end": 7852
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7844,
              "end": 7852
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 7857,
                "end": 7865
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7857,
              "end": 7865
            }
          }
        ],
        "loc": {
          "start": 7736,
          "end": 7867
        }
      },
      "loc": {
        "start": 7732,
        "end": 7867
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7906,
          "end": 7908
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7906,
        "end": 7908
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7909,
          "end": 7918
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7909,
        "end": 7918
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7948,
          "end": 7950
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7948,
        "end": 7950
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7951,
          "end": 7961
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7951,
        "end": 7961
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 7962,
          "end": 7965
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7962,
        "end": 7965
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7966,
          "end": 7975
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7966,
        "end": 7975
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7976,
          "end": 7988
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7995,
                "end": 7997
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7995,
              "end": 7997
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 8002,
                "end": 8010
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8002,
              "end": 8010
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 8015,
                "end": 8026
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8015,
              "end": 8026
            }
          }
        ],
        "loc": {
          "start": 7989,
          "end": 8028
        }
      },
      "loc": {
        "start": 7976,
        "end": 8028
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8029,
          "end": 8032
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 8039,
                "end": 8044
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8039,
              "end": 8044
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8049,
                "end": 8061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8049,
              "end": 8061
            }
          }
        ],
        "loc": {
          "start": 8033,
          "end": 8063
        }
      },
      "loc": {
        "start": 8029,
        "end": 8063
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 8095,
          "end": 8107
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8114,
                "end": 8116
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8114,
              "end": 8116
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 8121,
                "end": 8129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8121,
              "end": 8129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 8134,
                "end": 8137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8134,
              "end": 8137
            }
          }
        ],
        "loc": {
          "start": 8108,
          "end": 8139
        }
      },
      "loc": {
        "start": 8095,
        "end": 8139
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8140,
          "end": 8142
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8140,
        "end": 8142
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8143,
          "end": 8153
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8143,
        "end": 8153
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8154,
          "end": 8160
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8154,
        "end": 8160
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8161,
          "end": 8166
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8161,
        "end": 8166
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8167,
          "end": 8171
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8167,
        "end": 8171
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 8172,
          "end": 8181
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8172,
        "end": 8181
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 8182,
          "end": 8202
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8182,
        "end": 8202
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8203,
          "end": 8206
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 8213,
                "end": 8222
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8213,
              "end": 8222
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 8227,
                "end": 8236
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8227,
              "end": 8236
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 8241,
                "end": 8250
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8241,
              "end": 8250
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8255,
                "end": 8267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8255,
              "end": 8267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 8272,
                "end": 8280
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8272,
              "end": 8280
            }
          }
        ],
        "loc": {
          "start": 8207,
          "end": 8282
        }
      },
      "loc": {
        "start": 8203,
        "end": 8282
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8313,
          "end": 8315
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8313,
        "end": 8315
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8316,
          "end": 8321
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8316,
        "end": 8321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8322,
          "end": 8326
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8322,
        "end": 8326
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8327,
          "end": 8333
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8327,
        "end": 8333
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Api_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_list",
        "loc": {
          "start": 10,
          "end": 18
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 22,
            "end": 25
          }
        },
        "loc": {
          "start": 22,
          "end": 25
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 28,
                "end": 36
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 43,
                      "end": 55
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 66,
                            "end": 68
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 66,
                          "end": 68
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 77,
                            "end": 85
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 77,
                          "end": 85
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 94,
                            "end": 101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 94,
                          "end": 101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 110,
                            "end": 114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 123,
                            "end": 130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 123,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 56,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 43,
                    "end": 136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 141,
                      "end": 143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 141,
                    "end": 143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 148,
                      "end": 158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 148,
                    "end": 158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 163,
                      "end": 173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 178,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 178,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 191,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 209,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 209,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 231,
                      "end": 241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 246,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 246,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 259,
                      "end": 268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 259,
                    "end": 268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 273,
                      "end": 285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 273,
                    "end": 285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 290,
                      "end": 302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 290,
                    "end": 302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 307,
                      "end": 319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 307,
                    "end": 319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 324,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 338,
                            "end": 348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 338,
                          "end": 348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 357,
                            "end": 364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 357,
                          "end": 364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 373,
                            "end": 382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 373,
                          "end": 382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 391,
                            "end": 400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 391,
                          "end": 400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 409,
                            "end": 418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 409,
                          "end": 418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 427,
                            "end": 433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 427,
                          "end": 433
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 442,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 442,
                          "end": 449
                        }
                      }
                    ],
                    "loc": {
                      "start": 328,
                      "end": 455
                    }
                  },
                  "loc": {
                    "start": 324,
                    "end": 455
                  }
                }
              ],
              "loc": {
                "start": 37,
                "end": 457
              }
            },
            "loc": {
              "start": 28,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 458,
                "end": 460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 461,
                "end": 471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 461,
              "end": 471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 472,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 472,
              "end": 482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 483,
                "end": 492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 483,
              "end": 492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 493,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 493,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 505,
                "end": 511
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 521,
                      "end": 531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 531
                  }
                }
              ],
              "loc": {
                "start": 512,
                "end": 533
              }
            },
            "loc": {
              "start": 505,
              "end": 533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 534,
                "end": 539
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 553,
                        "end": 565
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 565
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 579,
                            "end": 595
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 576,
                          "end": 595
                        }
                      }
                    ],
                    "loc": {
                      "start": 566,
                      "end": 601
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 601
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 613,
                        "end": 617
                      }
                    },
                    "loc": {
                      "start": 613,
                      "end": 617
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 631,
                            "end": 639
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 628,
                          "end": 639
                        }
                      }
                    ],
                    "loc": {
                      "start": 618,
                      "end": 645
                    }
                  },
                  "loc": {
                    "start": 606,
                    "end": 645
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 647
              }
            },
            "loc": {
              "start": 534,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 660,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 660,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 675,
                "end": 680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 675,
              "end": 680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 681,
                "end": 690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 681,
              "end": 690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 691,
                "end": 695
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 705,
                      "end": 713
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 702,
                    "end": 713
                  }
                }
              ],
              "loc": {
                "start": 696,
                "end": 715
              }
            },
            "loc": {
              "start": 691,
              "end": 715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 716,
                "end": 730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 716,
              "end": 730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 731,
                "end": 736
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 736
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 737,
                "end": 740
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 747,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 747,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 761,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 777,
                      "end": 788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 793,
                      "end": 802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 793,
                    "end": 802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 807,
                      "end": 814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 807,
                    "end": 814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 819,
                      "end": 827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 819,
                    "end": 827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 832,
                      "end": 844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 832,
                    "end": 844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 849,
                      "end": 857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 849,
                    "end": 857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 862,
                      "end": 870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 862,
                    "end": 870
                  }
                }
              ],
              "loc": {
                "start": 741,
                "end": 872
              }
            },
            "loc": {
              "start": 737,
              "end": 872
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 874
        }
      },
      "loc": {
        "start": 1,
        "end": 874
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 884,
          "end": 891
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 895,
            "end": 898
          }
        },
        "loc": {
          "start": 895,
          "end": 898
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 901,
                "end": 903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 901,
              "end": 903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 904,
                "end": 913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 904,
              "end": 913
            }
          }
        ],
        "loc": {
          "start": 899,
          "end": 915
        }
      },
      "loc": {
        "start": 875,
        "end": 915
      }
    },
    "Label_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_full",
        "loc": {
          "start": 925,
          "end": 935
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 939,
            "end": 944
          }
        },
        "loc": {
          "start": 939,
          "end": 944
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apisCount",
              "loc": {
                "start": 947,
                "end": 956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 947,
              "end": 956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModesCount",
              "loc": {
                "start": 957,
                "end": 972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 957,
              "end": 972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 973,
                "end": 984
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 973,
              "end": 984
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetingsCount",
              "loc": {
                "start": 985,
                "end": 998
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 985,
              "end": 998
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 999,
                "end": 1009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 999,
              "end": 1009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 1010,
                "end": 1023
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1010,
              "end": 1023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 1024,
                "end": 1037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1024,
              "end": 1037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedulesCount",
              "loc": {
                "start": 1038,
                "end": 1052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1038,
              "end": 1052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 1053,
                "end": 1072
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1053,
              "end": 1072
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 1073,
                "end": 1087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1073,
              "end": 1087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1088,
                "end": 1090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1088,
              "end": 1090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1091,
                "end": 1101
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1091,
              "end": 1101
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1102,
                "end": 1112
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1102,
              "end": 1112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1113,
                "end": 1118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1113,
              "end": 1118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1119,
                "end": 1124
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1119,
              "end": 1124
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1125,
                "end": 1130
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1144,
                        "end": 1156
                      }
                    },
                    "loc": {
                      "start": 1144,
                      "end": 1156
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1170,
                            "end": 1186
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1167,
                          "end": 1186
                        }
                      }
                    ],
                    "loc": {
                      "start": 1157,
                      "end": 1192
                    }
                  },
                  "loc": {
                    "start": 1137,
                    "end": 1192
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1204,
                        "end": 1208
                      }
                    },
                    "loc": {
                      "start": 1204,
                      "end": 1208
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1222,
                            "end": 1230
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1219,
                          "end": 1230
                        }
                      }
                    ],
                    "loc": {
                      "start": 1209,
                      "end": 1236
                    }
                  },
                  "loc": {
                    "start": 1197,
                    "end": 1236
                  }
                }
              ],
              "loc": {
                "start": 1131,
                "end": 1238
              }
            },
            "loc": {
              "start": 1125,
              "end": 1238
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1239,
                "end": 1242
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1249,
                      "end": 1258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1249,
                    "end": 1258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1263,
                      "end": 1272
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1263,
                    "end": 1272
                  }
                }
              ],
              "loc": {
                "start": 1243,
                "end": 1274
              }
            },
            "loc": {
              "start": 1239,
              "end": 1274
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1276
        }
      },
      "loc": {
        "start": 916,
        "end": 1276
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 1286,
          "end": 1296
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 1300,
            "end": 1305
          }
        },
        "loc": {
          "start": 1300,
          "end": 1305
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1308,
                "end": 1310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1308,
              "end": 1310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1311,
                "end": 1321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1311,
              "end": 1321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1322,
                "end": 1332
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1322,
              "end": 1332
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1333,
                "end": 1338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1333,
              "end": 1338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1339,
                "end": 1344
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1339,
              "end": 1344
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1345,
                "end": 1350
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1364,
                        "end": 1376
                      }
                    },
                    "loc": {
                      "start": 1364,
                      "end": 1376
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1390,
                            "end": 1406
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1387,
                          "end": 1406
                        }
                      }
                    ],
                    "loc": {
                      "start": 1377,
                      "end": 1412
                    }
                  },
                  "loc": {
                    "start": 1357,
                    "end": 1412
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1424,
                        "end": 1428
                      }
                    },
                    "loc": {
                      "start": 1424,
                      "end": 1428
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1442,
                            "end": 1450
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1439,
                          "end": 1450
                        }
                      }
                    ],
                    "loc": {
                      "start": 1429,
                      "end": 1456
                    }
                  },
                  "loc": {
                    "start": 1417,
                    "end": 1456
                  }
                }
              ],
              "loc": {
                "start": 1351,
                "end": 1458
              }
            },
            "loc": {
              "start": 1345,
              "end": 1458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1459,
                "end": 1462
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1469,
                      "end": 1478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1469,
                    "end": 1478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1483,
                      "end": 1492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1483,
                    "end": 1492
                  }
                }
              ],
              "loc": {
                "start": 1463,
                "end": 1494
              }
            },
            "loc": {
              "start": 1459,
              "end": 1494
            }
          }
        ],
        "loc": {
          "start": 1306,
          "end": 1496
        }
      },
      "loc": {
        "start": 1277,
        "end": 1496
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 1506,
          "end": 1515
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1519,
            "end": 1523
          }
        },
        "loc": {
          "start": 1519,
          "end": 1523
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 1526,
                "end": 1534
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 1541,
                      "end": 1553
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1564,
                            "end": 1566
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1564,
                          "end": 1566
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1575,
                            "end": 1583
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1575,
                          "end": 1583
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1592,
                            "end": 1603
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1592,
                          "end": 1603
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1612,
                            "end": 1616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1612,
                          "end": 1616
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 1625,
                            "end": 1629
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1625,
                          "end": 1629
                        }
                      }
                    ],
                    "loc": {
                      "start": 1554,
                      "end": 1635
                    }
                  },
                  "loc": {
                    "start": 1541,
                    "end": 1635
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1640,
                      "end": 1642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1640,
                    "end": 1642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1647,
                      "end": 1657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1647,
                    "end": 1657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1662,
                      "end": 1672
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1662,
                    "end": 1672
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1677,
                      "end": 1685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1677,
                    "end": 1685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1690,
                      "end": 1699
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1690,
                    "end": 1699
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1704,
                      "end": 1716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1704,
                    "end": 1716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1721,
                      "end": 1733
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1721,
                    "end": 1733
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1738,
                      "end": 1750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1738,
                    "end": 1750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1755,
                      "end": 1758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 1769,
                            "end": 1779
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1769,
                          "end": 1779
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1788,
                            "end": 1795
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1788,
                          "end": 1795
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1804,
                            "end": 1813
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1804,
                          "end": 1813
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1822,
                            "end": 1831
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1822,
                          "end": 1831
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1840,
                            "end": 1849
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1840,
                          "end": 1849
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1858,
                            "end": 1864
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1858,
                          "end": 1864
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1873,
                            "end": 1880
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1873,
                          "end": 1880
                        }
                      }
                    ],
                    "loc": {
                      "start": 1759,
                      "end": 1886
                    }
                  },
                  "loc": {
                    "start": 1755,
                    "end": 1886
                  }
                }
              ],
              "loc": {
                "start": 1535,
                "end": 1888
              }
            },
            "loc": {
              "start": 1526,
              "end": 1888
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1889,
                "end": 1891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1889,
              "end": 1891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1892,
                "end": 1902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1892,
              "end": 1902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1903,
                "end": 1913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1903,
              "end": 1913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1914,
                "end": 1923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1914,
              "end": 1923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1924,
                "end": 1935
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1924,
              "end": 1935
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1936,
                "end": 1942
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 1952,
                      "end": 1962
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1949,
                    "end": 1962
                  }
                }
              ],
              "loc": {
                "start": 1943,
                "end": 1964
              }
            },
            "loc": {
              "start": 1936,
              "end": 1964
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1965,
                "end": 1970
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1984,
                        "end": 1996
                      }
                    },
                    "loc": {
                      "start": 1984,
                      "end": 1996
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 2010,
                            "end": 2026
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2007,
                          "end": 2026
                        }
                      }
                    ],
                    "loc": {
                      "start": 1997,
                      "end": 2032
                    }
                  },
                  "loc": {
                    "start": 1977,
                    "end": 2032
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2044,
                        "end": 2048
                      }
                    },
                    "loc": {
                      "start": 2044,
                      "end": 2048
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2062,
                            "end": 2070
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2059,
                          "end": 2070
                        }
                      }
                    ],
                    "loc": {
                      "start": 2049,
                      "end": 2076
                    }
                  },
                  "loc": {
                    "start": 2037,
                    "end": 2076
                  }
                }
              ],
              "loc": {
                "start": 1971,
                "end": 2078
              }
            },
            "loc": {
              "start": 1965,
              "end": 2078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2079,
                "end": 2090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2079,
              "end": 2090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2091,
                "end": 2105
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2091,
              "end": 2105
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2106,
                "end": 2111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2106,
              "end": 2111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2112,
                "end": 2121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2112,
              "end": 2121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2122,
                "end": 2126
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2136,
                      "end": 2144
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2133,
                    "end": 2144
                  }
                }
              ],
              "loc": {
                "start": 2127,
                "end": 2146
              }
            },
            "loc": {
              "start": 2122,
              "end": 2146
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2147,
                "end": 2161
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2147,
              "end": 2161
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2162,
                "end": 2167
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2162,
              "end": 2167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2168,
                "end": 2171
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2178,
                      "end": 2187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2178,
                    "end": 2187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2192,
                      "end": 2203
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2192,
                    "end": 2203
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2208,
                      "end": 2219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2208,
                    "end": 2219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2224,
                      "end": 2233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2224,
                    "end": 2233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2238,
                      "end": 2245
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2238,
                    "end": 2245
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2250,
                      "end": 2258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2250,
                    "end": 2258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2263,
                      "end": 2275
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2263,
                    "end": 2275
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2280,
                      "end": 2288
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2280,
                    "end": 2288
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2293,
                      "end": 2301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2293,
                    "end": 2301
                  }
                }
              ],
              "loc": {
                "start": 2172,
                "end": 2303
              }
            },
            "loc": {
              "start": 2168,
              "end": 2303
            }
          }
        ],
        "loc": {
          "start": 1524,
          "end": 2305
        }
      },
      "loc": {
        "start": 1497,
        "end": 2305
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2315,
          "end": 2323
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2327,
            "end": 2331
          }
        },
        "loc": {
          "start": 2327,
          "end": 2331
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2334,
                "end": 2336
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2334,
              "end": 2336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2337,
                "end": 2346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2337,
              "end": 2346
            }
          }
        ],
        "loc": {
          "start": 2332,
          "end": 2348
        }
      },
      "loc": {
        "start": 2306,
        "end": 2348
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 2358,
          "end": 2375
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2379,
            "end": 2391
          }
        },
        "loc": {
          "start": 2379,
          "end": 2391
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2394,
                "end": 2396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2394,
              "end": 2396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2397,
                "end": 2403
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2397,
              "end": 2403
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2404,
                "end": 2414
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2404,
              "end": 2414
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2415,
                "end": 2425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2415,
              "end": 2425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 2426,
                "end": 2444
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2426,
              "end": 2444
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2445,
                "end": 2454
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2445,
              "end": 2454
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2455,
                "end": 2468
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2455,
              "end": 2468
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 2469,
                "end": 2481
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2469,
              "end": 2481
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2482,
                "end": 2494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2482,
              "end": 2494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2495,
                "end": 2504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2495,
              "end": 2504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2505,
                "end": 2509
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 2519,
                      "end": 2527
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2516,
                    "end": 2527
                  }
                }
              ],
              "loc": {
                "start": 2510,
                "end": 2529
              }
            },
            "loc": {
              "start": 2505,
              "end": 2529
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2530,
                "end": 2542
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2549,
                      "end": 2551
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2549,
                    "end": 2551
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2556,
                      "end": 2564
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2556,
                    "end": 2564
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 2569,
                      "end": 2572
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2569,
                    "end": 2572
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2577,
                      "end": 2581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2577,
                    "end": 2581
                  }
                }
              ],
              "loc": {
                "start": 2543,
                "end": 2583
              }
            },
            "loc": {
              "start": 2530,
              "end": 2583
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2584,
                "end": 2587
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2594,
                      "end": 2607
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2594,
                    "end": 2607
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2612,
                      "end": 2621
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2612,
                    "end": 2621
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2626,
                      "end": 2637
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2626,
                    "end": 2637
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2642,
                      "end": 2651
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2642,
                    "end": 2651
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2656,
                      "end": 2665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2656,
                    "end": 2665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2670,
                      "end": 2677
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2670,
                    "end": 2677
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2682,
                      "end": 2694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2682,
                    "end": 2694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2699,
                      "end": 2707
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2699,
                    "end": 2707
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2712,
                      "end": 2726
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2737,
                            "end": 2739
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2737,
                          "end": 2739
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2748,
                            "end": 2758
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2748,
                          "end": 2758
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2767,
                            "end": 2777
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2767,
                          "end": 2777
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2786,
                            "end": 2793
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2786,
                          "end": 2793
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2802,
                            "end": 2813
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2802,
                          "end": 2813
                        }
                      }
                    ],
                    "loc": {
                      "start": 2727,
                      "end": 2819
                    }
                  },
                  "loc": {
                    "start": 2712,
                    "end": 2819
                  }
                }
              ],
              "loc": {
                "start": 2588,
                "end": 2821
              }
            },
            "loc": {
              "start": 2584,
              "end": 2821
            }
          }
        ],
        "loc": {
          "start": 2392,
          "end": 2823
        }
      },
      "loc": {
        "start": 2349,
        "end": 2823
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 2833,
          "end": 2849
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2853,
            "end": 2865
          }
        },
        "loc": {
          "start": 2853,
          "end": 2865
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2868,
                "end": 2870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2868,
              "end": 2870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2871,
                "end": 2877
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2871,
              "end": 2877
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2878,
                "end": 2881
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2888,
                      "end": 2901
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2888,
                    "end": 2901
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2906,
                      "end": 2915
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2906,
                    "end": 2915
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2920,
                      "end": 2931
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2920,
                    "end": 2931
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2936,
                      "end": 2945
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2936,
                    "end": 2945
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2950,
                      "end": 2959
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2950,
                    "end": 2959
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2964,
                      "end": 2971
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2964,
                    "end": 2971
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2976,
                      "end": 2988
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2976,
                    "end": 2988
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2993,
                      "end": 3001
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2993,
                    "end": 3001
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 3006,
                      "end": 3020
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3031,
                            "end": 3033
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3031,
                          "end": 3033
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3042,
                            "end": 3052
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3042,
                          "end": 3052
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3061,
                            "end": 3071
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3061,
                          "end": 3071
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3080,
                            "end": 3087
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3080,
                          "end": 3087
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3096,
                            "end": 3107
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3096,
                          "end": 3107
                        }
                      }
                    ],
                    "loc": {
                      "start": 3021,
                      "end": 3113
                    }
                  },
                  "loc": {
                    "start": 3006,
                    "end": 3113
                  }
                }
              ],
              "loc": {
                "start": 2882,
                "end": 3115
              }
            },
            "loc": {
              "start": 2878,
              "end": 3115
            }
          }
        ],
        "loc": {
          "start": 2866,
          "end": 3117
        }
      },
      "loc": {
        "start": 2824,
        "end": 3117
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 3127,
          "end": 3139
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3143,
            "end": 3150
          }
        },
        "loc": {
          "start": 3143,
          "end": 3150
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 3153,
                "end": 3161
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3168,
                      "end": 3180
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3191,
                            "end": 3193
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3191,
                          "end": 3193
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3202,
                            "end": 3210
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3202,
                          "end": 3210
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3219,
                            "end": 3230
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3219,
                          "end": 3230
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3239,
                            "end": 3243
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3239,
                          "end": 3243
                        }
                      }
                    ],
                    "loc": {
                      "start": 3181,
                      "end": 3249
                    }
                  },
                  "loc": {
                    "start": 3168,
                    "end": 3249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3254,
                      "end": 3256
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3254,
                    "end": 3256
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3261,
                      "end": 3271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3261,
                    "end": 3271
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3276,
                      "end": 3286
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3276,
                    "end": 3286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3291,
                      "end": 3307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3291,
                    "end": 3307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3312,
                      "end": 3320
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3312,
                    "end": 3320
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3325,
                      "end": 3334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3325,
                    "end": 3334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3339,
                      "end": 3351
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3339,
                    "end": 3351
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3356,
                      "end": 3372
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3356,
                    "end": 3372
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3377,
                      "end": 3387
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3377,
                    "end": 3387
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3392,
                      "end": 3404
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3392,
                    "end": 3404
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3409,
                      "end": 3421
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3409,
                    "end": 3421
                  }
                }
              ],
              "loc": {
                "start": 3162,
                "end": 3423
              }
            },
            "loc": {
              "start": 3153,
              "end": 3423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3424,
                "end": 3426
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3424,
              "end": 3426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3427,
                "end": 3437
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3427,
              "end": 3437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3438,
                "end": 3448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3438,
              "end": 3448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3449,
                "end": 3458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3449,
              "end": 3458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3459,
                "end": 3470
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3459,
              "end": 3470
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3471,
                "end": 3477
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 3487,
                      "end": 3497
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3484,
                    "end": 3497
                  }
                }
              ],
              "loc": {
                "start": 3478,
                "end": 3499
              }
            },
            "loc": {
              "start": 3471,
              "end": 3499
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3500,
                "end": 3505
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 3519,
                        "end": 3531
                      }
                    },
                    "loc": {
                      "start": 3519,
                      "end": 3531
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 3545,
                            "end": 3561
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3542,
                          "end": 3561
                        }
                      }
                    ],
                    "loc": {
                      "start": 3532,
                      "end": 3567
                    }
                  },
                  "loc": {
                    "start": 3512,
                    "end": 3567
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 3579,
                        "end": 3583
                      }
                    },
                    "loc": {
                      "start": 3579,
                      "end": 3583
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 3597,
                            "end": 3605
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3594,
                          "end": 3605
                        }
                      }
                    ],
                    "loc": {
                      "start": 3584,
                      "end": 3611
                    }
                  },
                  "loc": {
                    "start": 3572,
                    "end": 3611
                  }
                }
              ],
              "loc": {
                "start": 3506,
                "end": 3613
              }
            },
            "loc": {
              "start": 3500,
              "end": 3613
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3614,
                "end": 3625
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3614,
              "end": 3625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3626,
                "end": 3640
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3626,
              "end": 3640
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3641,
                "end": 3646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3641,
              "end": 3646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3647,
                "end": 3656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3647,
              "end": 3656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3657,
                "end": 3661
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 3671,
                      "end": 3679
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3668,
                    "end": 3679
                  }
                }
              ],
              "loc": {
                "start": 3662,
                "end": 3681
              }
            },
            "loc": {
              "start": 3657,
              "end": 3681
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3682,
                "end": 3696
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3682,
              "end": 3696
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3697,
                "end": 3702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3697,
              "end": 3702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3703,
                "end": 3706
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 3713,
                      "end": 3722
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3713,
                    "end": 3722
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3727,
                      "end": 3738
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3727,
                    "end": 3738
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3743,
                      "end": 3754
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3743,
                    "end": 3754
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3759,
                      "end": 3768
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3759,
                    "end": 3768
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3773,
                      "end": 3780
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3773,
                    "end": 3780
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3785,
                      "end": 3793
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3785,
                    "end": 3793
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3798,
                      "end": 3810
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3798,
                    "end": 3810
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3815,
                      "end": 3823
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3815,
                    "end": 3823
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3828,
                      "end": 3836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3828,
                    "end": 3836
                  }
                }
              ],
              "loc": {
                "start": 3707,
                "end": 3838
              }
            },
            "loc": {
              "start": 3703,
              "end": 3838
            }
          }
        ],
        "loc": {
          "start": 3151,
          "end": 3840
        }
      },
      "loc": {
        "start": 3118,
        "end": 3840
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3850,
          "end": 3861
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3865,
            "end": 3872
          }
        },
        "loc": {
          "start": 3865,
          "end": 3872
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3875,
                "end": 3877
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3875,
              "end": 3877
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3878,
                "end": 3887
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3878,
              "end": 3887
            }
          }
        ],
        "loc": {
          "start": 3873,
          "end": 3889
        }
      },
      "loc": {
        "start": 3841,
        "end": 3889
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3899,
          "end": 3912
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3916,
            "end": 3924
          }
        },
        "loc": {
          "start": 3916,
          "end": 3924
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3927,
                "end": 3939
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3946,
                      "end": 3948
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3946,
                    "end": 3948
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3953,
                      "end": 3961
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3953,
                    "end": 3961
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3966,
                      "end": 3977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3966,
                    "end": 3977
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3982,
                      "end": 3986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3982,
                    "end": 3986
                  }
                }
              ],
              "loc": {
                "start": 3940,
                "end": 3988
              }
            },
            "loc": {
              "start": 3927,
              "end": 3988
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3989,
                "end": 3991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3989,
              "end": 3991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3992,
                "end": 4002
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3992,
              "end": 4002
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4003,
                "end": 4013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4003,
              "end": 4013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 4014,
                "end": 4023
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4030,
                      "end": 4032
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4030,
                    "end": 4032
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 4037,
                      "end": 4042
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4037,
                    "end": 4042
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4047,
                      "end": 4051
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4047,
                    "end": 4051
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 4056,
                      "end": 4062
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4056,
                    "end": 4062
                  }
                }
              ],
              "loc": {
                "start": 4024,
                "end": 4064
              }
            },
            "loc": {
              "start": 4014,
              "end": 4064
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 4065,
                "end": 4082
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4065,
              "end": 4082
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4083,
                "end": 4092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4083,
              "end": 4092
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4093,
                "end": 4098
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4093,
              "end": 4098
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4099,
                "end": 4108
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4099,
              "end": 4108
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4109,
                "end": 4121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4109,
              "end": 4121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4122,
                "end": 4135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4122,
              "end": 4135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4136,
                "end": 4148
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4136,
              "end": 4148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4149,
                "end": 4158
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Api",
                      "loc": {
                        "start": 4172,
                        "end": 4175
                      }
                    },
                    "loc": {
                      "start": 4172,
                      "end": 4175
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Api_nav",
                          "loc": {
                            "start": 4189,
                            "end": 4196
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4186,
                          "end": 4196
                        }
                      }
                    ],
                    "loc": {
                      "start": 4176,
                      "end": 4202
                    }
                  },
                  "loc": {
                    "start": 4165,
                    "end": 4202
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Note",
                      "loc": {
                        "start": 4214,
                        "end": 4218
                      }
                    },
                    "loc": {
                      "start": 4214,
                      "end": 4218
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Note_nav",
                          "loc": {
                            "start": 4232,
                            "end": 4240
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4229,
                          "end": 4240
                        }
                      }
                    ],
                    "loc": {
                      "start": 4219,
                      "end": 4246
                    }
                  },
                  "loc": {
                    "start": 4207,
                    "end": 4246
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 4258,
                        "end": 4270
                      }
                    },
                    "loc": {
                      "start": 4258,
                      "end": 4270
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 4284,
                            "end": 4300
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4281,
                          "end": 4300
                        }
                      }
                    ],
                    "loc": {
                      "start": 4271,
                      "end": 4306
                    }
                  },
                  "loc": {
                    "start": 4251,
                    "end": 4306
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Project",
                      "loc": {
                        "start": 4318,
                        "end": 4325
                      }
                    },
                    "loc": {
                      "start": 4318,
                      "end": 4325
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Project_nav",
                          "loc": {
                            "start": 4339,
                            "end": 4350
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4336,
                          "end": 4350
                        }
                      }
                    ],
                    "loc": {
                      "start": 4326,
                      "end": 4356
                    }
                  },
                  "loc": {
                    "start": 4311,
                    "end": 4356
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Routine",
                      "loc": {
                        "start": 4368,
                        "end": 4375
                      }
                    },
                    "loc": {
                      "start": 4368,
                      "end": 4375
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Routine_nav",
                          "loc": {
                            "start": 4389,
                            "end": 4400
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4386,
                          "end": 4400
                        }
                      }
                    ],
                    "loc": {
                      "start": 4376,
                      "end": 4406
                    }
                  },
                  "loc": {
                    "start": 4361,
                    "end": 4406
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "SmartContract",
                      "loc": {
                        "start": 4418,
                        "end": 4431
                      }
                    },
                    "loc": {
                      "start": 4418,
                      "end": 4431
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "SmartContract_nav",
                          "loc": {
                            "start": 4445,
                            "end": 4462
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4442,
                          "end": 4462
                        }
                      }
                    ],
                    "loc": {
                      "start": 4432,
                      "end": 4468
                    }
                  },
                  "loc": {
                    "start": 4411,
                    "end": 4468
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Standard",
                      "loc": {
                        "start": 4480,
                        "end": 4488
                      }
                    },
                    "loc": {
                      "start": 4480,
                      "end": 4488
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Standard_nav",
                          "loc": {
                            "start": 4502,
                            "end": 4514
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4499,
                          "end": 4514
                        }
                      }
                    ],
                    "loc": {
                      "start": 4489,
                      "end": 4520
                    }
                  },
                  "loc": {
                    "start": 4473,
                    "end": 4520
                  }
                }
              ],
              "loc": {
                "start": 4159,
                "end": 4522
              }
            },
            "loc": {
              "start": 4149,
              "end": 4522
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4523,
                "end": 4527
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 4537,
                      "end": 4545
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4534,
                    "end": 4545
                  }
                }
              ],
              "loc": {
                "start": 4528,
                "end": 4547
              }
            },
            "loc": {
              "start": 4523,
              "end": 4547
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4548,
                "end": 4551
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 4558,
                      "end": 4566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4558,
                    "end": 4566
                  }
                }
              ],
              "loc": {
                "start": 4552,
                "end": 4568
              }
            },
            "loc": {
              "start": 4548,
              "end": 4568
            }
          }
        ],
        "loc": {
          "start": 3925,
          "end": 4570
        }
      },
      "loc": {
        "start": 3890,
        "end": 4570
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4580,
          "end": 4592
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4596,
            "end": 4603
          }
        },
        "loc": {
          "start": 4596,
          "end": 4603
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 4606,
                "end": 4614
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 4621,
                      "end": 4633
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4644,
                            "end": 4646
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4644,
                          "end": 4646
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4655,
                            "end": 4663
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4655,
                          "end": 4663
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4672,
                            "end": 4683
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4672,
                          "end": 4683
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4692,
                            "end": 4704
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4692,
                          "end": 4704
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4713,
                            "end": 4717
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4713,
                          "end": 4717
                        }
                      }
                    ],
                    "loc": {
                      "start": 4634,
                      "end": 4723
                    }
                  },
                  "loc": {
                    "start": 4621,
                    "end": 4723
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4728,
                      "end": 4730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4728,
                    "end": 4730
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4735,
                      "end": 4745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4735,
                    "end": 4745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4750,
                      "end": 4760
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4750,
                    "end": 4760
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4765,
                      "end": 4776
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4765,
                    "end": 4776
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4781,
                      "end": 4794
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4781,
                    "end": 4794
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4799,
                      "end": 4809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4799,
                    "end": 4809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4814,
                      "end": 4823
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4814,
                    "end": 4823
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4828,
                      "end": 4836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4828,
                    "end": 4836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4841,
                      "end": 4850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4841,
                    "end": 4850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4855,
                      "end": 4865
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4855,
                    "end": 4865
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4870,
                      "end": 4882
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4870,
                    "end": 4882
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4887,
                      "end": 4901
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4887,
                    "end": 4901
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 4906,
                      "end": 4927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4906,
                    "end": 4927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 4932,
                      "end": 4943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4932,
                    "end": 4943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4948,
                      "end": 4960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4948,
                    "end": 4960
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4965,
                      "end": 4977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4965,
                    "end": 4977
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4982,
                      "end": 4995
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4982,
                    "end": 4995
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5000,
                      "end": 5022
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5000,
                    "end": 5022
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5027,
                      "end": 5037
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5027,
                    "end": 5037
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 5042,
                      "end": 5053
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5042,
                    "end": 5053
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 5058,
                      "end": 5068
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5058,
                    "end": 5068
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 5073,
                      "end": 5087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5073,
                    "end": 5087
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 5092,
                      "end": 5104
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5092,
                    "end": 5104
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5109,
                      "end": 5121
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5109,
                    "end": 5121
                  }
                }
              ],
              "loc": {
                "start": 4615,
                "end": 5123
              }
            },
            "loc": {
              "start": 4606,
              "end": 5123
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5124,
                "end": 5126
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5124,
              "end": 5126
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5127,
                "end": 5137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5127,
              "end": 5137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5138,
                "end": 5148
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5138,
              "end": 5148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5149,
                "end": 5159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5149,
              "end": 5159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5160,
                "end": 5169
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5160,
              "end": 5169
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5170,
                "end": 5181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5170,
              "end": 5181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5182,
                "end": 5188
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 5198,
                      "end": 5208
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5195,
                    "end": 5208
                  }
                }
              ],
              "loc": {
                "start": 5189,
                "end": 5210
              }
            },
            "loc": {
              "start": 5182,
              "end": 5210
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5211,
                "end": 5216
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 5230,
                        "end": 5242
                      }
                    },
                    "loc": {
                      "start": 5230,
                      "end": 5242
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5256,
                            "end": 5272
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5253,
                          "end": 5272
                        }
                      }
                    ],
                    "loc": {
                      "start": 5243,
                      "end": 5278
                    }
                  },
                  "loc": {
                    "start": 5223,
                    "end": 5278
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 5290,
                        "end": 5294
                      }
                    },
                    "loc": {
                      "start": 5290,
                      "end": 5294
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5308,
                            "end": 5316
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5305,
                          "end": 5316
                        }
                      }
                    ],
                    "loc": {
                      "start": 5295,
                      "end": 5322
                    }
                  },
                  "loc": {
                    "start": 5283,
                    "end": 5322
                  }
                }
              ],
              "loc": {
                "start": 5217,
                "end": 5324
              }
            },
            "loc": {
              "start": 5211,
              "end": 5324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5325,
                "end": 5336
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5325,
              "end": 5336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5337,
                "end": 5351
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5337,
              "end": 5351
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5352,
                "end": 5357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5352,
              "end": 5357
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5358,
                "end": 5367
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5358,
              "end": 5367
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5368,
                "end": 5372
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 5382,
                      "end": 5390
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5379,
                    "end": 5390
                  }
                }
              ],
              "loc": {
                "start": 5373,
                "end": 5392
              }
            },
            "loc": {
              "start": 5368,
              "end": 5392
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5393,
                "end": 5407
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5393,
              "end": 5407
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5408,
                "end": 5413
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5408,
              "end": 5413
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5414,
                "end": 5417
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 5424,
                      "end": 5434
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5424,
                    "end": 5434
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5439,
                      "end": 5448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5439,
                    "end": 5448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5453,
                      "end": 5464
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5453,
                    "end": 5464
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5469,
                      "end": 5478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5469,
                    "end": 5478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5483,
                      "end": 5490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5483,
                    "end": 5490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5495,
                      "end": 5503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5495,
                    "end": 5503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5508,
                      "end": 5520
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5508,
                    "end": 5520
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5525,
                      "end": 5533
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5525,
                    "end": 5533
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5538,
                      "end": 5546
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5538,
                    "end": 5546
                  }
                }
              ],
              "loc": {
                "start": 5418,
                "end": 5548
              }
            },
            "loc": {
              "start": 5414,
              "end": 5548
            }
          }
        ],
        "loc": {
          "start": 4604,
          "end": 5550
        }
      },
      "loc": {
        "start": 4571,
        "end": 5550
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5560,
          "end": 5571
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5575,
            "end": 5582
          }
        },
        "loc": {
          "start": 5575,
          "end": 5582
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5585,
                "end": 5587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5585,
              "end": 5587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5588,
                "end": 5598
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5588,
              "end": 5598
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5599,
                "end": 5608
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5599,
              "end": 5608
            }
          }
        ],
        "loc": {
          "start": 5583,
          "end": 5610
        }
      },
      "loc": {
        "start": 5551,
        "end": 5610
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 5620,
          "end": 5635
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 5639,
            "end": 5647
          }
        },
        "loc": {
          "start": 5639,
          "end": 5647
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5650,
                "end": 5652
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5650,
              "end": 5652
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5653,
                "end": 5663
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5653,
              "end": 5663
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5664,
                "end": 5674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5664,
              "end": 5674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 5675,
                "end": 5684
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5675,
              "end": 5684
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 5685,
                "end": 5692
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5685,
              "end": 5692
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 5693,
                "end": 5701
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5693,
              "end": 5701
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 5702,
                "end": 5712
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5719,
                      "end": 5721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5719,
                    "end": 5721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 5726,
                      "end": 5743
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5726,
                    "end": 5743
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 5748,
                      "end": 5760
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5748,
                    "end": 5760
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 5765,
                      "end": 5775
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5765,
                    "end": 5775
                  }
                }
              ],
              "loc": {
                "start": 5713,
                "end": 5777
              }
            },
            "loc": {
              "start": 5702,
              "end": 5777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 5778,
                "end": 5789
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5796,
                      "end": 5798
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5796,
                    "end": 5798
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 5803,
                      "end": 5817
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5803,
                    "end": 5817
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 5822,
                      "end": 5830
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5822,
                    "end": 5830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 5835,
                      "end": 5844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5835,
                    "end": 5844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 5849,
                      "end": 5859
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5849,
                    "end": 5859
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 5864,
                      "end": 5869
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5864,
                    "end": 5869
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 5874,
                      "end": 5881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5874,
                    "end": 5881
                  }
                }
              ],
              "loc": {
                "start": 5790,
                "end": 5883
              }
            },
            "loc": {
              "start": 5778,
              "end": 5883
            }
          }
        ],
        "loc": {
          "start": 5648,
          "end": 5885
        }
      },
      "loc": {
        "start": 5611,
        "end": 5885
      }
    },
    "SmartContract_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_list",
        "loc": {
          "start": 5895,
          "end": 5913
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 5917,
            "end": 5930
          }
        },
        "loc": {
          "start": 5917,
          "end": 5930
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 5933,
                "end": 5941
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5948,
                      "end": 5960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5971,
                            "end": 5973
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5971,
                          "end": 5973
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5982,
                            "end": 5990
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5982,
                          "end": 5990
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5999,
                            "end": 6010
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5999,
                          "end": 6010
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6019,
                            "end": 6031
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6019,
                          "end": 6031
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6040,
                            "end": 6044
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6040,
                          "end": 6044
                        }
                      }
                    ],
                    "loc": {
                      "start": 5961,
                      "end": 6050
                    }
                  },
                  "loc": {
                    "start": 5948,
                    "end": 6050
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6055,
                      "end": 6057
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6055,
                    "end": 6057
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 6062,
                      "end": 6072
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6062,
                    "end": 6072
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 6077,
                      "end": 6087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6077,
                    "end": 6087
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6092,
                      "end": 6102
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6092,
                    "end": 6102
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 6107,
                      "end": 6116
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6107,
                    "end": 6116
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6121,
                      "end": 6129
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6121,
                    "end": 6129
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6134,
                      "end": 6143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6134,
                    "end": 6143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6148,
                      "end": 6155
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6148,
                    "end": 6155
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 6160,
                      "end": 6172
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6160,
                    "end": 6172
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 6177,
                      "end": 6184
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6177,
                    "end": 6184
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6189,
                      "end": 6201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6189,
                    "end": 6201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6206,
                      "end": 6218
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6206,
                    "end": 6218
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6223,
                      "end": 6236
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6223,
                    "end": 6236
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6241,
                      "end": 6263
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6241,
                    "end": 6263
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6268,
                      "end": 6278
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6268,
                    "end": 6278
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6283,
                      "end": 6295
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6283,
                    "end": 6295
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6300,
                      "end": 6303
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 6314,
                            "end": 6324
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6314,
                          "end": 6324
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6333,
                            "end": 6340
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6333,
                          "end": 6340
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6349,
                            "end": 6358
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6349,
                          "end": 6358
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6367,
                            "end": 6376
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6367,
                          "end": 6376
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6385,
                            "end": 6394
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6385,
                          "end": 6394
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6403,
                            "end": 6409
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6403,
                          "end": 6409
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6418,
                            "end": 6425
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6418,
                          "end": 6425
                        }
                      }
                    ],
                    "loc": {
                      "start": 6304,
                      "end": 6431
                    }
                  },
                  "loc": {
                    "start": 6300,
                    "end": 6431
                  }
                }
              ],
              "loc": {
                "start": 5942,
                "end": 6433
              }
            },
            "loc": {
              "start": 5933,
              "end": 6433
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6434,
                "end": 6436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6434,
              "end": 6436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6437,
                "end": 6447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6437,
              "end": 6447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6448,
                "end": 6458
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6448,
              "end": 6458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6459,
                "end": 6468
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6459,
              "end": 6468
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6469,
                "end": 6480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6469,
              "end": 6480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6481,
                "end": 6487
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 6497,
                      "end": 6507
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6494,
                    "end": 6507
                  }
                }
              ],
              "loc": {
                "start": 6488,
                "end": 6509
              }
            },
            "loc": {
              "start": 6481,
              "end": 6509
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6510,
                "end": 6515
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 6529,
                        "end": 6541
                      }
                    },
                    "loc": {
                      "start": 6529,
                      "end": 6541
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6555,
                            "end": 6571
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6552,
                          "end": 6571
                        }
                      }
                    ],
                    "loc": {
                      "start": 6542,
                      "end": 6577
                    }
                  },
                  "loc": {
                    "start": 6522,
                    "end": 6577
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 6589,
                        "end": 6593
                      }
                    },
                    "loc": {
                      "start": 6589,
                      "end": 6593
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6607,
                            "end": 6615
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6604,
                          "end": 6615
                        }
                      }
                    ],
                    "loc": {
                      "start": 6594,
                      "end": 6621
                    }
                  },
                  "loc": {
                    "start": 6582,
                    "end": 6621
                  }
                }
              ],
              "loc": {
                "start": 6516,
                "end": 6623
              }
            },
            "loc": {
              "start": 6510,
              "end": 6623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6624,
                "end": 6635
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6624,
              "end": 6635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6636,
                "end": 6650
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6636,
              "end": 6650
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6651,
                "end": 6656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6651,
              "end": 6656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6657,
                "end": 6666
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6657,
              "end": 6666
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6667,
                "end": 6671
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 6681,
                      "end": 6689
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6678,
                    "end": 6689
                  }
                }
              ],
              "loc": {
                "start": 6672,
                "end": 6691
              }
            },
            "loc": {
              "start": 6667,
              "end": 6691
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6692,
                "end": 6706
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6692,
              "end": 6706
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6707,
                "end": 6712
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6707,
              "end": 6712
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6713,
                "end": 6716
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6723,
                      "end": 6732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6723,
                    "end": 6732
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6737,
                      "end": 6748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6737,
                    "end": 6748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6753,
                      "end": 6764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6753,
                    "end": 6764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6769,
                      "end": 6778
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6769,
                    "end": 6778
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6783,
                      "end": 6790
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6783,
                    "end": 6790
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6795,
                      "end": 6803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6795,
                    "end": 6803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6808,
                      "end": 6820
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6808,
                    "end": 6820
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6825,
                      "end": 6833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6825,
                    "end": 6833
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6838,
                      "end": 6846
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6838,
                    "end": 6846
                  }
                }
              ],
              "loc": {
                "start": 6717,
                "end": 6848
              }
            },
            "loc": {
              "start": 6713,
              "end": 6848
            }
          }
        ],
        "loc": {
          "start": 5931,
          "end": 6850
        }
      },
      "loc": {
        "start": 5886,
        "end": 6850
      }
    },
    "SmartContract_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_nav",
        "loc": {
          "start": 6860,
          "end": 6877
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6881,
            "end": 6894
          }
        },
        "loc": {
          "start": 6881,
          "end": 6894
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6897,
                "end": 6899
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6897,
              "end": 6899
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6900,
                "end": 6909
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6900,
              "end": 6909
            }
          }
        ],
        "loc": {
          "start": 6895,
          "end": 6911
        }
      },
      "loc": {
        "start": 6851,
        "end": 6911
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 6921,
          "end": 6934
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6938,
            "end": 6946
          }
        },
        "loc": {
          "start": 6938,
          "end": 6946
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 6949,
                "end": 6957
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6964,
                      "end": 6976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6987,
                            "end": 6989
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6987,
                          "end": 6989
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6998,
                            "end": 7006
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6998,
                          "end": 7006
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7015,
                            "end": 7026
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7015,
                          "end": 7026
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 7035,
                            "end": 7047
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7035,
                          "end": 7047
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7056,
                            "end": 7060
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7056,
                          "end": 7060
                        }
                      }
                    ],
                    "loc": {
                      "start": 6977,
                      "end": 7066
                    }
                  },
                  "loc": {
                    "start": 6964,
                    "end": 7066
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7071,
                      "end": 7073
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7071,
                    "end": 7073
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7078,
                      "end": 7088
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7078,
                    "end": 7088
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7093,
                      "end": 7103
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7093,
                    "end": 7103
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 7108,
                      "end": 7118
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7108,
                    "end": 7118
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 7123,
                      "end": 7129
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7123,
                    "end": 7129
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 7134,
                      "end": 7142
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7134,
                    "end": 7142
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7147,
                      "end": 7156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7147,
                    "end": 7156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 7161,
                      "end": 7168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7161,
                    "end": 7168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 7173,
                      "end": 7185
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7173,
                    "end": 7185
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 7190,
                      "end": 7195
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7190,
                    "end": 7195
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 7200,
                      "end": 7203
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7200,
                    "end": 7203
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 7208,
                      "end": 7220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7208,
                    "end": 7220
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 7225,
                      "end": 7237
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7225,
                    "end": 7237
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 7242,
                      "end": 7255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7242,
                    "end": 7255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 7260,
                      "end": 7282
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7260,
                    "end": 7282
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 7287,
                      "end": 7297
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7287,
                    "end": 7297
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 7302,
                      "end": 7314
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7302,
                    "end": 7314
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7319,
                      "end": 7322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 7333,
                            "end": 7343
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7333,
                          "end": 7343
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 7352,
                            "end": 7359
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7352,
                          "end": 7359
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 7368,
                            "end": 7377
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7368,
                          "end": 7377
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 7386,
                            "end": 7395
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7386,
                          "end": 7395
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7404,
                            "end": 7413
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7404,
                          "end": 7413
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 7422,
                            "end": 7428
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7422,
                          "end": 7428
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7437,
                            "end": 7444
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7437,
                          "end": 7444
                        }
                      }
                    ],
                    "loc": {
                      "start": 7323,
                      "end": 7450
                    }
                  },
                  "loc": {
                    "start": 7319,
                    "end": 7450
                  }
                }
              ],
              "loc": {
                "start": 6958,
                "end": 7452
              }
            },
            "loc": {
              "start": 6949,
              "end": 7452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7453,
                "end": 7455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7453,
              "end": 7455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7456,
                "end": 7466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7456,
              "end": 7466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7467,
                "end": 7477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7467,
              "end": 7477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7478,
                "end": 7487
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7478,
              "end": 7487
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 7488,
                "end": 7499
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7488,
              "end": 7499
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 7500,
                "end": 7506
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 7516,
                      "end": 7526
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7513,
                    "end": 7526
                  }
                }
              ],
              "loc": {
                "start": 7507,
                "end": 7528
              }
            },
            "loc": {
              "start": 7500,
              "end": 7528
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 7529,
                "end": 7534
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 7548,
                        "end": 7560
                      }
                    },
                    "loc": {
                      "start": 7548,
                      "end": 7560
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7574,
                            "end": 7590
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7571,
                          "end": 7590
                        }
                      }
                    ],
                    "loc": {
                      "start": 7561,
                      "end": 7596
                    }
                  },
                  "loc": {
                    "start": 7541,
                    "end": 7596
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 7608,
                        "end": 7612
                      }
                    },
                    "loc": {
                      "start": 7608,
                      "end": 7612
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 7626,
                            "end": 7634
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7623,
                          "end": 7634
                        }
                      }
                    ],
                    "loc": {
                      "start": 7613,
                      "end": 7640
                    }
                  },
                  "loc": {
                    "start": 7601,
                    "end": 7640
                  }
                }
              ],
              "loc": {
                "start": 7535,
                "end": 7642
              }
            },
            "loc": {
              "start": 7529,
              "end": 7642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 7643,
                "end": 7654
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7643,
              "end": 7654
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 7655,
                "end": 7669
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7655,
              "end": 7669
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 7670,
                "end": 7675
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7670,
              "end": 7675
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7676,
                "end": 7685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7676,
              "end": 7685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 7686,
                "end": 7690
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 7700,
                      "end": 7708
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7697,
                    "end": 7708
                  }
                }
              ],
              "loc": {
                "start": 7691,
                "end": 7710
              }
            },
            "loc": {
              "start": 7686,
              "end": 7710
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 7711,
                "end": 7725
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7711,
              "end": 7725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 7726,
                "end": 7731
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7726,
              "end": 7731
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7732,
                "end": 7735
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7742,
                      "end": 7751
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7742,
                    "end": 7751
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7756,
                      "end": 7767
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7756,
                    "end": 7767
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 7772,
                      "end": 7783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7772,
                    "end": 7783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7788,
                      "end": 7797
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7788,
                    "end": 7797
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7802,
                      "end": 7809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7802,
                    "end": 7809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 7814,
                      "end": 7822
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7814,
                    "end": 7822
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7827,
                      "end": 7839
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7827,
                    "end": 7839
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7844,
                      "end": 7852
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7844,
                    "end": 7852
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 7857,
                      "end": 7865
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7857,
                    "end": 7865
                  }
                }
              ],
              "loc": {
                "start": 7736,
                "end": 7867
              }
            },
            "loc": {
              "start": 7732,
              "end": 7867
            }
          }
        ],
        "loc": {
          "start": 6947,
          "end": 7869
        }
      },
      "loc": {
        "start": 6912,
        "end": 7869
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 7879,
          "end": 7891
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7895,
            "end": 7903
          }
        },
        "loc": {
          "start": 7895,
          "end": 7903
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7906,
                "end": 7908
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7906,
              "end": 7908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7909,
                "end": 7918
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7909,
              "end": 7918
            }
          }
        ],
        "loc": {
          "start": 7904,
          "end": 7920
        }
      },
      "loc": {
        "start": 7870,
        "end": 7920
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 7930,
          "end": 7938
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 7942,
            "end": 7945
          }
        },
        "loc": {
          "start": 7942,
          "end": 7945
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7948,
                "end": 7950
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7948,
              "end": 7950
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7951,
                "end": 7961
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7951,
              "end": 7961
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 7962,
                "end": 7965
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7962,
              "end": 7965
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7966,
                "end": 7975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7966,
              "end": 7975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7976,
                "end": 7988
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7995,
                      "end": 7997
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7995,
                    "end": 7997
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 8002,
                      "end": 8010
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8002,
                    "end": 8010
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 8015,
                      "end": 8026
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8015,
                    "end": 8026
                  }
                }
              ],
              "loc": {
                "start": 7989,
                "end": 8028
              }
            },
            "loc": {
              "start": 7976,
              "end": 8028
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8029,
                "end": 8032
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 8039,
                      "end": 8044
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8039,
                    "end": 8044
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8049,
                      "end": 8061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8049,
                    "end": 8061
                  }
                }
              ],
              "loc": {
                "start": 8033,
                "end": 8063
              }
            },
            "loc": {
              "start": 8029,
              "end": 8063
            }
          }
        ],
        "loc": {
          "start": 7946,
          "end": 8065
        }
      },
      "loc": {
        "start": 7921,
        "end": 8065
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 8075,
          "end": 8084
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8088,
            "end": 8092
          }
        },
        "loc": {
          "start": 8088,
          "end": 8092
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 8095,
                "end": 8107
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8114,
                      "end": 8116
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8114,
                    "end": 8116
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 8121,
                      "end": 8129
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8121,
                    "end": 8129
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 8134,
                      "end": 8137
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8134,
                    "end": 8137
                  }
                }
              ],
              "loc": {
                "start": 8108,
                "end": 8139
              }
            },
            "loc": {
              "start": 8095,
              "end": 8139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8140,
                "end": 8142
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8140,
              "end": 8142
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8143,
                "end": 8153
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8143,
              "end": 8153
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8154,
                "end": 8160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8154,
              "end": 8160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8161,
                "end": 8166
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8161,
              "end": 8166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8167,
                "end": 8171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8167,
              "end": 8171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 8172,
                "end": 8181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8172,
              "end": 8181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 8182,
                "end": 8202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8182,
              "end": 8202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8203,
                "end": 8206
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 8213,
                      "end": 8222
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8213,
                    "end": 8222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 8227,
                      "end": 8236
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8227,
                    "end": 8236
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 8241,
                      "end": 8250
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8241,
                    "end": 8250
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8255,
                      "end": 8267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8255,
                    "end": 8267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 8272,
                      "end": 8280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8272,
                    "end": 8280
                  }
                }
              ],
              "loc": {
                "start": 8207,
                "end": 8282
              }
            },
            "loc": {
              "start": 8203,
              "end": 8282
            }
          }
        ],
        "loc": {
          "start": 8093,
          "end": 8284
        }
      },
      "loc": {
        "start": 8066,
        "end": 8284
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 8294,
          "end": 8302
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8306,
            "end": 8310
          }
        },
        "loc": {
          "start": 8306,
          "end": 8310
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8313,
                "end": 8315
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8313,
              "end": 8315
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8316,
                "end": 8321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8316,
              "end": 8321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8322,
                "end": 8326
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8322,
              "end": 8326
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8327,
                "end": 8333
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8327,
              "end": 8333
            }
          }
        ],
        "loc": {
          "start": 8311,
          "end": 8335
        }
      },
      "loc": {
        "start": 8285,
        "end": 8335
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "popular",
      "loc": {
        "start": 8343,
        "end": 8350
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8352,
              "end": 8357
            }
          },
          "loc": {
            "start": 8351,
            "end": 8357
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularInput",
              "loc": {
                "start": 8359,
                "end": 8371
              }
            },
            "loc": {
              "start": 8359,
              "end": 8371
            }
          },
          "loc": {
            "start": 8359,
            "end": 8372
          }
        },
        "directives": [],
        "loc": {
          "start": 8351,
          "end": 8372
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "popular",
            "loc": {
              "start": 8378,
              "end": 8385
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8386,
                  "end": 8391
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8394,
                    "end": 8399
                  }
                },
                "loc": {
                  "start": 8393,
                  "end": 8399
                }
              },
              "loc": {
                "start": 8386,
                "end": 8399
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "apis",
                  "loc": {
                    "start": 8407,
                    "end": 8411
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Api_list",
                        "loc": {
                          "start": 8425,
                          "end": 8433
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8422,
                        "end": 8433
                      }
                    }
                  ],
                  "loc": {
                    "start": 8412,
                    "end": 8439
                  }
                },
                "loc": {
                  "start": 8407,
                  "end": 8439
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "notes",
                  "loc": {
                    "start": 8444,
                    "end": 8449
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Note_list",
                        "loc": {
                          "start": 8463,
                          "end": 8472
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8460,
                        "end": 8472
                      }
                    }
                  ],
                  "loc": {
                    "start": 8450,
                    "end": 8478
                  }
                },
                "loc": {
                  "start": 8444,
                  "end": 8478
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "organizations",
                  "loc": {
                    "start": 8483,
                    "end": 8496
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Organization_list",
                        "loc": {
                          "start": 8510,
                          "end": 8527
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8507,
                        "end": 8527
                      }
                    }
                  ],
                  "loc": {
                    "start": 8497,
                    "end": 8533
                  }
                },
                "loc": {
                  "start": 8483,
                  "end": 8533
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "projects",
                  "loc": {
                    "start": 8538,
                    "end": 8546
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Project_list",
                        "loc": {
                          "start": 8560,
                          "end": 8572
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8557,
                        "end": 8572
                      }
                    }
                  ],
                  "loc": {
                    "start": 8547,
                    "end": 8578
                  }
                },
                "loc": {
                  "start": 8538,
                  "end": 8578
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "questions",
                  "loc": {
                    "start": 8583,
                    "end": 8592
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Question_list",
                        "loc": {
                          "start": 8606,
                          "end": 8619
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8603,
                        "end": 8619
                      }
                    }
                  ],
                  "loc": {
                    "start": 8593,
                    "end": 8625
                  }
                },
                "loc": {
                  "start": 8583,
                  "end": 8625
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "routines",
                  "loc": {
                    "start": 8630,
                    "end": 8638
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Routine_list",
                        "loc": {
                          "start": 8652,
                          "end": 8664
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8649,
                        "end": 8664
                      }
                    }
                  ],
                  "loc": {
                    "start": 8639,
                    "end": 8670
                  }
                },
                "loc": {
                  "start": 8630,
                  "end": 8670
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "smartContracts",
                  "loc": {
                    "start": 8675,
                    "end": 8689
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "SmartContract_list",
                        "loc": {
                          "start": 8703,
                          "end": 8721
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8700,
                        "end": 8721
                      }
                    }
                  ],
                  "loc": {
                    "start": 8690,
                    "end": 8727
                  }
                },
                "loc": {
                  "start": 8675,
                  "end": 8727
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "standards",
                  "loc": {
                    "start": 8732,
                    "end": 8741
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Standard_list",
                        "loc": {
                          "start": 8755,
                          "end": 8768
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8752,
                        "end": 8768
                      }
                    }
                  ],
                  "loc": {
                    "start": 8742,
                    "end": 8774
                  }
                },
                "loc": {
                  "start": 8732,
                  "end": 8774
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 8779,
                    "end": 8784
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "User_list",
                        "loc": {
                          "start": 8798,
                          "end": 8807
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8795,
                        "end": 8807
                      }
                    }
                  ],
                  "loc": {
                    "start": 8785,
                    "end": 8813
                  }
                },
                "loc": {
                  "start": 8779,
                  "end": 8813
                }
              }
            ],
            "loc": {
              "start": 8401,
              "end": 8817
            }
          },
          "loc": {
            "start": 8378,
            "end": 8817
          }
        }
      ],
      "loc": {
        "start": 8374,
        "end": 8819
      }
    },
    "loc": {
      "start": 8337,
      "end": 8819
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_popular"
  }
} as const;
