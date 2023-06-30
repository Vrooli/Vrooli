export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 6852,
          "end": 6856
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 6857,
              "end": 6862
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 6865,
                "end": 6870
              }
            },
            "loc": {
              "start": 6864,
              "end": 6870
            }
          },
          "loc": {
            "start": 6857,
            "end": 6870
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
              "value": "notes",
              "loc": {
                "start": 6878,
                "end": 6883
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
                      "start": 6897,
                      "end": 6906
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6894,
                    "end": 6906
                  }
                }
              ],
              "loc": {
                "start": 6884,
                "end": 6912
              }
            },
            "loc": {
              "start": 6878,
              "end": 6912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 6917,
                "end": 6926
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
                    "value": "Reminder_full",
                    "loc": {
                      "start": 6940,
                      "end": 6953
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6937,
                    "end": 6953
                  }
                }
              ],
              "loc": {
                "start": 6927,
                "end": 6959
              }
            },
            "loc": {
              "start": 6917,
              "end": 6959
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 6964,
                "end": 6973
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
                    "value": "Resource_list",
                    "loc": {
                      "start": 6987,
                      "end": 7000
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6984,
                    "end": 7000
                  }
                }
              ],
              "loc": {
                "start": 6974,
                "end": 7006
              }
            },
            "loc": {
              "start": 6964,
              "end": 7006
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 7011,
                "end": 7020
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
                    "value": "Schedule_list",
                    "loc": {
                      "start": 7034,
                      "end": 7047
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7031,
                    "end": 7047
                  }
                }
              ],
              "loc": {
                "start": 7021,
                "end": 7053
              }
            },
            "loc": {
              "start": 7011,
              "end": 7053
            }
          }
        ],
        "loc": {
          "start": 6872,
          "end": 7057
        }
      },
      "loc": {
        "start": 6852,
        "end": 7057
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
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
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
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
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
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
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
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
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
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
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 250,
          "end": 258
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
                "start": 265,
                "end": 277
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
                      "start": 288,
                      "end": 290
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 288,
                    "end": 290
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 299,
                      "end": 307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 299,
                    "end": 307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 316,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 316,
                    "end": 327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 336,
                      "end": 340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 336,
                    "end": 340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "text",
                    "loc": {
                      "start": 349,
                      "end": 353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 349,
                    "end": 353
                  }
                }
              ],
              "loc": {
                "start": 278,
                "end": 359
              }
            },
            "loc": {
              "start": 265,
              "end": 359
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 364,
                "end": 366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 364,
              "end": 366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 371,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 371,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 386,
                "end": 396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 386,
              "end": 396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 401,
                "end": 409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 401,
              "end": 409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 414,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 414,
              "end": 423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 428,
                "end": 440
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 428,
              "end": 440
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 445,
                "end": 457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 445,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 462,
                "end": 474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 462,
              "end": 474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 479,
                "end": 482
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
                      "start": 493,
                      "end": 503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 493,
                    "end": 503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 512,
                      "end": 519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 512,
                    "end": 519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 528,
                      "end": 537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 528,
                    "end": 537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 546,
                      "end": 555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 546,
                    "end": 555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 564,
                      "end": 573
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 564,
                    "end": 573
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 582,
                      "end": 588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 582,
                    "end": 588
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 597,
                      "end": 604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 597,
                    "end": 604
                  }
                }
              ],
              "loc": {
                "start": 483,
                "end": 610
              }
            },
            "loc": {
              "start": 479,
              "end": 610
            }
          }
        ],
        "loc": {
          "start": 259,
          "end": 612
        }
      },
      "loc": {
        "start": 250,
        "end": 612
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 613,
          "end": 615
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 613,
        "end": 615
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 616,
          "end": 626
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 616,
        "end": 626
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 627,
          "end": 637
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 627,
        "end": 637
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 638,
          "end": 647
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 638,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
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
        "value": "labels",
        "loc": {
          "start": 660,
          "end": 666
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
                "start": 676,
                "end": 686
              }
            },
            "directives": [],
            "loc": {
              "start": 673,
              "end": 686
            }
          }
        ],
        "loc": {
          "start": 667,
          "end": 688
        }
      },
      "loc": {
        "start": 660,
        "end": 688
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 689,
          "end": 694
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
                  "start": 708,
                  "end": 720
                }
              },
              "loc": {
                "start": 708,
                "end": 720
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
                      "start": 734,
                      "end": 750
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 731,
                    "end": 750
                  }
                }
              ],
              "loc": {
                "start": 721,
                "end": 756
              }
            },
            "loc": {
              "start": 701,
              "end": 756
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
                  "start": 768,
                  "end": 772
                }
              },
              "loc": {
                "start": 768,
                "end": 772
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
                      "start": 786,
                      "end": 794
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 783,
                    "end": 794
                  }
                }
              ],
              "loc": {
                "start": 773,
                "end": 800
              }
            },
            "loc": {
              "start": 761,
              "end": 800
            }
          }
        ],
        "loc": {
          "start": 695,
          "end": 802
        }
      },
      "loc": {
        "start": 689,
        "end": 802
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 803,
          "end": 814
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 803,
        "end": 814
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 815,
          "end": 829
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 815,
        "end": 829
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 830,
          "end": 835
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 830,
        "end": 835
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 836,
          "end": 845
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 836,
        "end": 845
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 846,
          "end": 850
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
                "start": 860,
                "end": 868
              }
            },
            "directives": [],
            "loc": {
              "start": 857,
              "end": 868
            }
          }
        ],
        "loc": {
          "start": 851,
          "end": 870
        }
      },
      "loc": {
        "start": 846,
        "end": 870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 871,
          "end": 885
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 871,
        "end": 885
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 886,
          "end": 891
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 886,
        "end": 891
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 892,
          "end": 895
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
                "start": 902,
                "end": 911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 902,
              "end": 911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 916,
                "end": 927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 916,
              "end": 927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 932,
                "end": 943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 932,
              "end": 943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 948,
                "end": 957
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 948,
              "end": 957
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 962,
                "end": 969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 962,
              "end": 969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 974,
                "end": 982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 974,
              "end": 982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 987,
                "end": 999
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 987,
              "end": 999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1004,
                "end": 1012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1004,
              "end": 1012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1017,
                "end": 1025
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1017,
              "end": 1025
            }
          }
        ],
        "loc": {
          "start": 896,
          "end": 1027
        }
      },
      "loc": {
        "start": 892,
        "end": 1027
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1074,
          "end": 1076
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1074,
        "end": 1076
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1077,
          "end": 1083
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1077,
        "end": 1083
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1084,
          "end": 1087
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
                "start": 1094,
                "end": 1107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1094,
              "end": 1107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1112,
                "end": 1121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1112,
              "end": 1121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1126,
                "end": 1137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1126,
              "end": 1137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1142,
                "end": 1151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1142,
              "end": 1151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1156,
                "end": 1165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1156,
              "end": 1165
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1170,
                "end": 1177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1170,
              "end": 1177
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1182,
                "end": 1194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1182,
              "end": 1194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1199,
                "end": 1207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1199,
              "end": 1207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1212,
                "end": 1226
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
                      "start": 1237,
                      "end": 1239
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1237,
                    "end": 1239
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1248,
                      "end": 1258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1248,
                    "end": 1258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1267,
                      "end": 1277
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1267,
                    "end": 1277
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1286,
                      "end": 1293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1286,
                    "end": 1293
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1302,
                      "end": 1313
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1302,
                    "end": 1313
                  }
                }
              ],
              "loc": {
                "start": 1227,
                "end": 1319
              }
            },
            "loc": {
              "start": 1212,
              "end": 1319
            }
          }
        ],
        "loc": {
          "start": 1088,
          "end": 1321
        }
      },
      "loc": {
        "start": 1084,
        "end": 1321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1361,
          "end": 1363
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1361,
        "end": 1363
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1364,
          "end": 1374
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1364,
        "end": 1374
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1375,
          "end": 1385
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1375,
        "end": 1385
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1386,
          "end": 1390
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1386,
        "end": 1390
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 1391,
          "end": 1402
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1391,
        "end": 1402
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 1403,
          "end": 1410
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1403,
        "end": 1410
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1411,
          "end": 1416
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1411,
        "end": 1416
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 1417,
          "end": 1427
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1417,
        "end": 1427
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 1428,
          "end": 1441
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
                "start": 1448,
                "end": 1450
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1448,
              "end": 1450
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1455,
                "end": 1465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1455,
              "end": 1465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1470,
                "end": 1480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1470,
              "end": 1480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1485,
                "end": 1489
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1485,
              "end": 1489
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1494,
                "end": 1505
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1494,
              "end": 1505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1510,
                "end": 1517
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1510,
              "end": 1517
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1522,
                "end": 1527
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1522,
              "end": 1527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1532,
                "end": 1542
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1532,
              "end": 1542
            }
          }
        ],
        "loc": {
          "start": 1442,
          "end": 1544
        }
      },
      "loc": {
        "start": 1428,
        "end": 1544
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 1545,
          "end": 1557
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
              "value": "created_at",
              "loc": {
                "start": 1571,
                "end": 1581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1571,
              "end": 1581
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1586,
                "end": 1596
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1586,
              "end": 1596
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 1601,
                "end": 1610
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
                    "value": "labels",
                    "loc": {
                      "start": 1621,
                      "end": 1627
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
                            "start": 1642,
                            "end": 1644
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1642,
                          "end": 1644
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 1657,
                            "end": 1662
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1657,
                          "end": 1662
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 1675,
                            "end": 1680
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1675,
                          "end": 1680
                        }
                      }
                    ],
                    "loc": {
                      "start": 1628,
                      "end": 1690
                    }
                  },
                  "loc": {
                    "start": 1621,
                    "end": 1690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 1699,
                      "end": 1707
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
                          "value": "Schedule_common",
                          "loc": {
                            "start": 1725,
                            "end": 1740
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1722,
                          "end": 1740
                        }
                      }
                    ],
                    "loc": {
                      "start": 1708,
                      "end": 1750
                    }
                  },
                  "loc": {
                    "start": 1699,
                    "end": 1750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1759,
                      "end": 1761
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1759,
                    "end": 1761
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1770,
                      "end": 1774
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1770,
                    "end": 1774
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1783,
                      "end": 1794
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1783,
                    "end": 1794
                  }
                }
              ],
              "loc": {
                "start": 1611,
                "end": 1800
              }
            },
            "loc": {
              "start": 1601,
              "end": 1800
            }
          }
        ],
        "loc": {
          "start": 1558,
          "end": 1802
        }
      },
      "loc": {
        "start": 1545,
        "end": 1802
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1842,
          "end": 1844
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1842,
        "end": 1844
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1845,
          "end": 1850
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1845,
        "end": 1850
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1851,
          "end": 1855
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1851,
        "end": 1855
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1856,
          "end": 1863
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1856,
        "end": 1863
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1864,
          "end": 1876
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
                "start": 1883,
                "end": 1885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1883,
              "end": 1885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1890,
                "end": 1898
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1890,
              "end": 1898
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1903,
                "end": 1914
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1903,
              "end": 1914
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1919,
                "end": 1923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1919,
              "end": 1923
            }
          }
        ],
        "loc": {
          "start": 1877,
          "end": 1925
        }
      },
      "loc": {
        "start": 1864,
        "end": 1925
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1967,
          "end": 1969
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1967,
        "end": 1969
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1970,
          "end": 1980
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1970,
        "end": 1980
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1981,
          "end": 1991
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1981,
        "end": 1991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 1992,
          "end": 2001
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1992,
        "end": 2001
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 2002,
          "end": 2009
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2002,
        "end": 2009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 2010,
          "end": 2018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2010,
        "end": 2018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 2019,
          "end": 2029
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
                "start": 2036,
                "end": 2038
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2036,
              "end": 2038
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 2043,
                "end": 2060
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2043,
              "end": 2060
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 2065,
                "end": 2077
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2065,
              "end": 2077
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 2082,
                "end": 2092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2082,
              "end": 2092
            }
          }
        ],
        "loc": {
          "start": 2030,
          "end": 2094
        }
      },
      "loc": {
        "start": 2019,
        "end": 2094
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 2095,
          "end": 2106
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
                "start": 2113,
                "end": 2115
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2113,
              "end": 2115
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 2120,
                "end": 2134
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2120,
              "end": 2134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 2139,
                "end": 2147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2139,
              "end": 2147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 2152,
                "end": 2161
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2152,
              "end": 2161
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 2166,
                "end": 2176
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2166,
              "end": 2176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 2181,
                "end": 2186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2181,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 2191,
                "end": 2198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2191,
              "end": 2198
            }
          }
        ],
        "loc": {
          "start": 2107,
          "end": 2200
        }
      },
      "loc": {
        "start": 2095,
        "end": 2200
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2240,
          "end": 2246
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
                "start": 2256,
                "end": 2266
              }
            },
            "directives": [],
            "loc": {
              "start": 2253,
              "end": 2266
            }
          }
        ],
        "loc": {
          "start": 2247,
          "end": 2268
        }
      },
      "loc": {
        "start": 2240,
        "end": 2268
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 2269,
          "end": 2279
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
              "value": "labels",
              "loc": {
                "start": 2286,
                "end": 2292
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
                      "start": 2303,
                      "end": 2305
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2303,
                    "end": 2305
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 2314,
                      "end": 2319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2314,
                    "end": 2319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2328,
                      "end": 2333
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2328,
                    "end": 2333
                  }
                }
              ],
              "loc": {
                "start": 2293,
                "end": 2339
              }
            },
            "loc": {
              "start": 2286,
              "end": 2339
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2344,
                "end": 2346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2344,
              "end": 2346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2351,
                "end": 2355
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2351,
              "end": 2355
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2360,
                "end": 2371
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2360,
              "end": 2371
            }
          }
        ],
        "loc": {
          "start": 2280,
          "end": 2373
        }
      },
      "loc": {
        "start": 2269,
        "end": 2373
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 2374,
          "end": 2382
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
              "value": "labels",
              "loc": {
                "start": 2389,
                "end": 2395
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
                      "start": 2409,
                      "end": 2419
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2406,
                    "end": 2419
                  }
                }
              ],
              "loc": {
                "start": 2396,
                "end": 2425
              }
            },
            "loc": {
              "start": 2389,
              "end": 2425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2430,
                "end": 2442
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
                      "start": 2453,
                      "end": 2455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2453,
                    "end": 2455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2464,
                      "end": 2472
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2464,
                    "end": 2472
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2481,
                      "end": 2492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2481,
                    "end": 2492
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 2501,
                      "end": 2505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2501,
                    "end": 2505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2514,
                      "end": 2518
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2514,
                    "end": 2518
                  }
                }
              ],
              "loc": {
                "start": 2443,
                "end": 2524
              }
            },
            "loc": {
              "start": 2430,
              "end": 2524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2529,
                "end": 2531
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2529,
              "end": 2531
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 2536,
                "end": 2558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2536,
              "end": 2558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnOrganizationProfile",
              "loc": {
                "start": 2563,
                "end": 2588
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2563,
              "end": 2588
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 2593,
                "end": 2605
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
                      "start": 2616,
                      "end": 2618
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2616,
                    "end": 2618
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 2627,
                      "end": 2633
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2627,
                    "end": 2633
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2642,
                      "end": 2645
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
                            "start": 2660,
                            "end": 2673
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2660,
                          "end": 2673
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2686,
                            "end": 2695
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2686,
                          "end": 2695
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 2708,
                            "end": 2719
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2708,
                          "end": 2719
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2732,
                            "end": 2741
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2732,
                          "end": 2741
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2754,
                            "end": 2763
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2754,
                          "end": 2763
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2776,
                            "end": 2783
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2776,
                          "end": 2783
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 2796,
                            "end": 2808
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2796,
                          "end": 2808
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 2821,
                            "end": 2829
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2821,
                          "end": 2829
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 2842,
                            "end": 2856
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
                                  "start": 2875,
                                  "end": 2877
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2875,
                                "end": 2877
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2894,
                                  "end": 2904
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2894,
                                "end": 2904
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 2921,
                                  "end": 2931
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2921,
                                "end": 2931
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 2948,
                                  "end": 2955
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2948,
                                "end": 2955
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 2972,
                                  "end": 2983
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2972,
                                "end": 2983
                              }
                            }
                          ],
                          "loc": {
                            "start": 2857,
                            "end": 2997
                          }
                        },
                        "loc": {
                          "start": 2842,
                          "end": 2997
                        }
                      }
                    ],
                    "loc": {
                      "start": 2646,
                      "end": 3007
                    }
                  },
                  "loc": {
                    "start": 2642,
                    "end": 3007
                  }
                }
              ],
              "loc": {
                "start": 2606,
                "end": 3013
              }
            },
            "loc": {
              "start": 2593,
              "end": 3013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 3018,
                "end": 3035
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
                    "value": "members",
                    "loc": {
                      "start": 3046,
                      "end": 3053
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
                            "start": 3068,
                            "end": 3070
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3068,
                          "end": 3070
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3083,
                            "end": 3093
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3083,
                          "end": 3093
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3106,
                            "end": 3116
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3106,
                          "end": 3116
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3129,
                            "end": 3136
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3129,
                          "end": 3136
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3149,
                            "end": 3160
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3149,
                          "end": 3160
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 3173,
                            "end": 3178
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
                                  "start": 3197,
                                  "end": 3199
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3197,
                                "end": 3199
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3216,
                                  "end": 3226
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3216,
                                "end": 3226
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3243,
                                  "end": 3253
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3243,
                                "end": 3253
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3270,
                                  "end": 3274
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3270,
                                "end": 3274
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3291,
                                  "end": 3302
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3291,
                                "end": 3302
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 3319,
                                  "end": 3331
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3319,
                                "end": 3331
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 3348,
                                  "end": 3360
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
                                        "start": 3383,
                                        "end": 3385
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3383,
                                      "end": 3385
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 3406,
                                        "end": 3412
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3406,
                                      "end": 3412
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3433,
                                        "end": 3436
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
                                              "start": 3463,
                                              "end": 3476
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3463,
                                            "end": 3476
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 3501,
                                              "end": 3510
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3501,
                                            "end": 3510
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 3535,
                                              "end": 3546
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3535,
                                            "end": 3546
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 3571,
                                              "end": 3580
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3571,
                                            "end": 3580
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 3605,
                                              "end": 3614
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3605,
                                            "end": 3614
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 3639,
                                              "end": 3646
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3639,
                                            "end": 3646
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3671,
                                              "end": 3683
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3671,
                                            "end": 3683
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 3708,
                                              "end": 3716
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3708,
                                            "end": 3716
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 3741,
                                              "end": 3755
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
                                                    "start": 3786,
                                                    "end": 3788
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3786,
                                                  "end": 3788
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3817,
                                                    "end": 3827
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3817,
                                                  "end": 3827
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3856,
                                                    "end": 3866
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3856,
                                                  "end": 3866
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 3895,
                                                    "end": 3902
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3895,
                                                  "end": 3902
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 3931,
                                                    "end": 3942
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3931,
                                                  "end": 3942
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3756,
                                              "end": 3968
                                            }
                                          },
                                          "loc": {
                                            "start": 3741,
                                            "end": 3968
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3437,
                                        "end": 3990
                                      }
                                    },
                                    "loc": {
                                      "start": 3433,
                                      "end": 3990
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3361,
                                  "end": 4008
                                }
                              },
                              "loc": {
                                "start": 3348,
                                "end": 4008
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4025,
                                  "end": 4037
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
                                        "start": 4060,
                                        "end": 4062
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4060,
                                      "end": 4062
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4083,
                                        "end": 4091
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4083,
                                      "end": 4091
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4112,
                                        "end": 4123
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4112,
                                      "end": 4123
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4038,
                                  "end": 4141
                                }
                              },
                              "loc": {
                                "start": 4025,
                                "end": 4141
                              }
                            }
                          ],
                          "loc": {
                            "start": 3179,
                            "end": 4155
                          }
                        },
                        "loc": {
                          "start": 3173,
                          "end": 4155
                        }
                      }
                    ],
                    "loc": {
                      "start": 3054,
                      "end": 4165
                    }
                  },
                  "loc": {
                    "start": 3046,
                    "end": 4165
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4174,
                      "end": 4176
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4174,
                    "end": 4176
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4185,
                      "end": 4195
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4185,
                    "end": 4195
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4204,
                      "end": 4214
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4204,
                    "end": 4214
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4223,
                      "end": 4227
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4223,
                    "end": 4227
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 4236,
                      "end": 4247
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4236,
                    "end": 4247
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 4256,
                      "end": 4268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4256,
                    "end": 4268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 4277,
                      "end": 4289
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
                            "start": 4304,
                            "end": 4306
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4304,
                          "end": 4306
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 4319,
                            "end": 4325
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4319,
                          "end": 4325
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4338,
                            "end": 4341
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
                                  "start": 4360,
                                  "end": 4373
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4360,
                                "end": 4373
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 4390,
                                  "end": 4399
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4390,
                                "end": 4399
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 4416,
                                  "end": 4427
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4416,
                                "end": 4427
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 4444,
                                  "end": 4453
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4444,
                                "end": 4453
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4470,
                                  "end": 4479
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4470,
                                "end": 4479
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 4496,
                                  "end": 4503
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4496,
                                "end": 4503
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 4520,
                                  "end": 4532
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4520,
                                "end": 4532
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 4549,
                                  "end": 4557
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4549,
                                "end": 4557
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 4574,
                                  "end": 4588
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
                                        "start": 4611,
                                        "end": 4613
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4611,
                                      "end": 4613
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4634,
                                        "end": 4644
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4634,
                                      "end": 4644
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4665,
                                        "end": 4675
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4665,
                                      "end": 4675
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 4696,
                                        "end": 4703
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4696,
                                      "end": 4703
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4724,
                                        "end": 4735
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4724,
                                      "end": 4735
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4589,
                                  "end": 4753
                                }
                              },
                              "loc": {
                                "start": 4574,
                                "end": 4753
                              }
                            }
                          ],
                          "loc": {
                            "start": 4342,
                            "end": 4767
                          }
                        },
                        "loc": {
                          "start": 4338,
                          "end": 4767
                        }
                      }
                    ],
                    "loc": {
                      "start": 4290,
                      "end": 4777
                    }
                  },
                  "loc": {
                    "start": 4277,
                    "end": 4777
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 4786,
                      "end": 4798
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
                            "start": 4813,
                            "end": 4815
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4813,
                          "end": 4815
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
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
                          "value": "description",
                          "loc": {
                            "start": 4849,
                            "end": 4860
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4849,
                          "end": 4860
                        }
                      }
                    ],
                    "loc": {
                      "start": 4799,
                      "end": 4870
                    }
                  },
                  "loc": {
                    "start": 4786,
                    "end": 4870
                  }
                }
              ],
              "loc": {
                "start": 3036,
                "end": 4876
              }
            },
            "loc": {
              "start": 3018,
              "end": 4876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 4881,
                "end": 4895
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4881,
              "end": 4895
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 4900,
                "end": 4912
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4900,
              "end": 4912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4917,
                "end": 4920
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
                      "start": 4931,
                      "end": 4940
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4931,
                    "end": 4940
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 4949,
                      "end": 4958
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4949,
                    "end": 4958
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 4967,
                      "end": 4976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4967,
                    "end": 4976
                  }
                }
              ],
              "loc": {
                "start": 4921,
                "end": 4982
              }
            },
            "loc": {
              "start": 4917,
              "end": 4982
            }
          }
        ],
        "loc": {
          "start": 2383,
          "end": 4984
        }
      },
      "loc": {
        "start": 2374,
        "end": 4984
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 4985,
          "end": 4996
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
              "value": "projectVersion",
              "loc": {
                "start": 5003,
                "end": 5017
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
                      "start": 5028,
                      "end": 5030
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5028,
                    "end": 5030
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5039,
                      "end": 5049
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5039,
                    "end": 5049
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5058,
                      "end": 5066
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5058,
                    "end": 5066
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5075,
                      "end": 5084
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5075,
                    "end": 5084
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5093,
                      "end": 5105
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5093,
                    "end": 5105
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5114,
                      "end": 5126
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5114,
                    "end": 5126
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 5135,
                      "end": 5139
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
                            "start": 5154,
                            "end": 5156
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5154,
                          "end": 5156
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5169,
                            "end": 5178
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5169,
                          "end": 5178
                        }
                      }
                    ],
                    "loc": {
                      "start": 5140,
                      "end": 5188
                    }
                  },
                  "loc": {
                    "start": 5135,
                    "end": 5188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5197,
                      "end": 5209
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
                            "start": 5224,
                            "end": 5226
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5224,
                          "end": 5226
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5239,
                            "end": 5247
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5239,
                          "end": 5247
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5260,
                            "end": 5271
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5260,
                          "end": 5271
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5284,
                            "end": 5288
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5284,
                          "end": 5288
                        }
                      }
                    ],
                    "loc": {
                      "start": 5210,
                      "end": 5298
                    }
                  },
                  "loc": {
                    "start": 5197,
                    "end": 5298
                  }
                }
              ],
              "loc": {
                "start": 5018,
                "end": 5304
              }
            },
            "loc": {
              "start": 5003,
              "end": 5304
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5309,
                "end": 5311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5309,
              "end": 5311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5316,
                "end": 5325
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5316,
              "end": 5325
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 5330,
                "end": 5349
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5330,
              "end": 5349
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 5354,
                "end": 5369
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5354,
              "end": 5369
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 5374,
                "end": 5383
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5374,
              "end": 5383
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 5388,
                "end": 5399
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5388,
              "end": 5399
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 5404,
                "end": 5415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5404,
              "end": 5415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 5420,
                "end": 5424
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5420,
              "end": 5424
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 5429,
                "end": 5435
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5429,
              "end": 5435
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 5440,
                "end": 5450
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5440,
              "end": 5450
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 5455,
                "end": 5467
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 5481,
                      "end": 5497
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5478,
                    "end": 5497
                  }
                }
              ],
              "loc": {
                "start": 5468,
                "end": 5503
              }
            },
            "loc": {
              "start": 5455,
              "end": 5503
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 5508,
                "end": 5512
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
                    "value": "User_nav",
                    "loc": {
                      "start": 5526,
                      "end": 5534
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5523,
                    "end": 5534
                  }
                }
              ],
              "loc": {
                "start": 5513,
                "end": 5540
              }
            },
            "loc": {
              "start": 5508,
              "end": 5540
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5545,
                "end": 5548
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
                      "start": 5559,
                      "end": 5568
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5559,
                    "end": 5568
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5577,
                      "end": 5586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5577,
                    "end": 5586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5595,
                      "end": 5602
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5595,
                    "end": 5602
                  }
                }
              ],
              "loc": {
                "start": 5549,
                "end": 5608
              }
            },
            "loc": {
              "start": 5545,
              "end": 5608
            }
          }
        ],
        "loc": {
          "start": 4997,
          "end": 5610
        }
      },
      "loc": {
        "start": 4985,
        "end": 5610
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 5611,
          "end": 5622
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
              "value": "routineVersion",
              "loc": {
                "start": 5629,
                "end": 5643
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
                      "start": 5654,
                      "end": 5656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5654,
                    "end": 5656
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5665,
                      "end": 5675
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5665,
                    "end": 5675
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 5684,
                      "end": 5697
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5684,
                    "end": 5697
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5706,
                      "end": 5716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5706,
                    "end": 5716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5725,
                      "end": 5734
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5725,
                    "end": 5734
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5743,
                      "end": 5751
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5743,
                    "end": 5751
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5760,
                      "end": 5769
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5760,
                    "end": 5769
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 5778,
                      "end": 5782
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
                            "start": 5797,
                            "end": 5799
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5797,
                          "end": 5799
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 5812,
                            "end": 5822
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5812,
                          "end": 5822
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
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
                      }
                    ],
                    "loc": {
                      "start": 5783,
                      "end": 5854
                    }
                  },
                  "loc": {
                    "start": 5778,
                    "end": 5854
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5863,
                      "end": 5875
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
                            "start": 5890,
                            "end": 5892
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5890,
                          "end": 5892
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5905,
                            "end": 5913
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5905,
                          "end": 5913
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5926,
                            "end": 5937
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5926,
                          "end": 5937
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 5950,
                            "end": 5962
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5950,
                          "end": 5962
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5975,
                            "end": 5979
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5975,
                          "end": 5979
                        }
                      }
                    ],
                    "loc": {
                      "start": 5876,
                      "end": 5989
                    }
                  },
                  "loc": {
                    "start": 5863,
                    "end": 5989
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5998,
                      "end": 6010
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5998,
                    "end": 6010
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
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
                }
              ],
              "loc": {
                "start": 5644,
                "end": 6037
              }
            },
            "loc": {
              "start": 5629,
              "end": 6037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6042,
                "end": 6044
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6042,
              "end": 6044
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6049,
                "end": 6058
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6049,
              "end": 6058
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6063,
                "end": 6082
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6063,
              "end": 6082
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6087,
                "end": 6102
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6087,
              "end": 6102
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
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
              "value": "timeElapsed",
              "loc": {
                "start": 6121,
                "end": 6132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6121,
              "end": 6132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6137,
                "end": 6148
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6137,
              "end": 6148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6153,
                "end": 6157
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6153,
              "end": 6157
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6162,
                "end": 6168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6162,
              "end": 6168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6173,
                "end": 6183
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6173,
              "end": 6183
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 6188,
                "end": 6199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6188,
              "end": 6199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 6204,
                "end": 6223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6204,
              "end": 6223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 6228,
                "end": 6240
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6254,
                      "end": 6270
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6251,
                    "end": 6270
                  }
                }
              ],
              "loc": {
                "start": 6241,
                "end": 6276
              }
            },
            "loc": {
              "start": 6228,
              "end": 6276
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6281,
                "end": 6285
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
                    "value": "User_nav",
                    "loc": {
                      "start": 6299,
                      "end": 6307
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6296,
                    "end": 6307
                  }
                }
              ],
              "loc": {
                "start": 6286,
                "end": 6313
              }
            },
            "loc": {
              "start": 6281,
              "end": 6313
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6318,
                "end": 6321
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
                      "start": 6332,
                      "end": 6341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6332,
                    "end": 6341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6350,
                      "end": 6359
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6350,
                    "end": 6359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6368,
                      "end": 6375
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6368,
                    "end": 6375
                  }
                }
              ],
              "loc": {
                "start": 6322,
                "end": 6381
              }
            },
            "loc": {
              "start": 6318,
              "end": 6381
            }
          }
        ],
        "loc": {
          "start": 5623,
          "end": 6383
        }
      },
      "loc": {
        "start": 5611,
        "end": 6383
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6384,
          "end": 6386
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6384,
        "end": 6386
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6387,
          "end": 6397
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6387,
        "end": 6397
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6398,
          "end": 6408
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6398,
        "end": 6408
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 6409,
          "end": 6418
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6409,
        "end": 6418
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 6419,
          "end": 6426
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6419,
        "end": 6426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 6427,
          "end": 6435
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6427,
        "end": 6435
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 6436,
          "end": 6446
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
                "start": 6453,
                "end": 6455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6453,
              "end": 6455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 6460,
                "end": 6477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6460,
              "end": 6477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 6482,
                "end": 6494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6482,
              "end": 6494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 6499,
                "end": 6509
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6499,
              "end": 6509
            }
          }
        ],
        "loc": {
          "start": 6447,
          "end": 6511
        }
      },
      "loc": {
        "start": 6436,
        "end": 6511
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 6512,
          "end": 6523
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
                "start": 6530,
                "end": 6532
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6530,
              "end": 6532
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 6537,
                "end": 6551
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6537,
              "end": 6551
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 6556,
                "end": 6564
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6556,
              "end": 6564
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 6569,
                "end": 6578
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6569,
              "end": 6578
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 6583,
                "end": 6593
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6583,
              "end": 6593
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 6598,
                "end": 6603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6598,
              "end": 6603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 6608,
                "end": 6615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6608,
              "end": 6615
            }
          }
        ],
        "loc": {
          "start": 6524,
          "end": 6617
        }
      },
      "loc": {
        "start": 6512,
        "end": 6617
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6647,
          "end": 6649
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6647,
        "end": 6649
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6650,
          "end": 6660
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6650,
        "end": 6660
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 6661,
          "end": 6664
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6661,
        "end": 6664
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6665,
          "end": 6674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6665,
        "end": 6674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6675,
          "end": 6687
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
                "start": 6694,
                "end": 6696
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6694,
              "end": 6696
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6701,
                "end": 6709
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6701,
              "end": 6709
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 6714,
                "end": 6725
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6714,
              "end": 6725
            }
          }
        ],
        "loc": {
          "start": 6688,
          "end": 6727
        }
      },
      "loc": {
        "start": 6675,
        "end": 6727
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6728,
          "end": 6731
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
                "start": 6738,
                "end": 6743
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6738,
              "end": 6743
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6748,
                "end": 6760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6748,
              "end": 6760
            }
          }
        ],
        "loc": {
          "start": 6732,
          "end": 6762
        }
      },
      "loc": {
        "start": 6728,
        "end": 6762
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6793,
          "end": 6795
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6793,
        "end": 6795
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 6796,
          "end": 6801
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6796,
        "end": 6801
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 6802,
          "end": 6806
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6802,
        "end": 6806
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 6807,
          "end": 6813
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6807,
        "end": 6813
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
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
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
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
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
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
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
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
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
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
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
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
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 230,
          "end": 239
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 243,
            "end": 247
          }
        },
        "loc": {
          "start": 243,
          "end": 247
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
                "start": 250,
                "end": 258
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
                      "start": 265,
                      "end": 277
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
                            "start": 288,
                            "end": 290
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 288,
                          "end": 290
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 299,
                            "end": 307
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 299,
                          "end": 307
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 316,
                            "end": 327
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 316,
                          "end": 327
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 336,
                            "end": 340
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 336,
                          "end": 340
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 349,
                            "end": 353
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 349,
                          "end": 353
                        }
                      }
                    ],
                    "loc": {
                      "start": 278,
                      "end": 359
                    }
                  },
                  "loc": {
                    "start": 265,
                    "end": 359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 364,
                      "end": 366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 364,
                    "end": 366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 371,
                      "end": 381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 371,
                    "end": 381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 386,
                      "end": 396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 386,
                    "end": 396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 401,
                      "end": 409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 401,
                    "end": 409
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 414,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 414,
                    "end": 423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 428,
                      "end": 440
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 428,
                    "end": 440
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 445,
                      "end": 457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 445,
                    "end": 457
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 462,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 462,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 479,
                      "end": 482
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
                            "start": 493,
                            "end": 503
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 493,
                          "end": 503
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 512,
                            "end": 519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 512,
                          "end": 519
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 528,
                            "end": 537
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 528,
                          "end": 537
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 546,
                            "end": 555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 546,
                          "end": 555
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 564,
                            "end": 573
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 564,
                          "end": 573
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 582,
                            "end": 588
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 582,
                          "end": 588
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 597,
                            "end": 604
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 597,
                          "end": 604
                        }
                      }
                    ],
                    "loc": {
                      "start": 483,
                      "end": 610
                    }
                  },
                  "loc": {
                    "start": 479,
                    "end": 610
                  }
                }
              ],
              "loc": {
                "start": 259,
                "end": 612
              }
            },
            "loc": {
              "start": 250,
              "end": 612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 613,
                "end": 615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 613,
              "end": 615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 616,
                "end": 626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 616,
              "end": 626
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 627,
                "end": 637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 627,
              "end": 637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 638,
                "end": 647
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 638,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
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
              "value": "labels",
              "loc": {
                "start": 660,
                "end": 666
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
                      "start": 676,
                      "end": 686
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 673,
                    "end": 686
                  }
                }
              ],
              "loc": {
                "start": 667,
                "end": 688
              }
            },
            "loc": {
              "start": 660,
              "end": 688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 689,
                "end": 694
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
                        "start": 708,
                        "end": 720
                      }
                    },
                    "loc": {
                      "start": 708,
                      "end": 720
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
                            "start": 734,
                            "end": 750
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 731,
                          "end": 750
                        }
                      }
                    ],
                    "loc": {
                      "start": 721,
                      "end": 756
                    }
                  },
                  "loc": {
                    "start": 701,
                    "end": 756
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
                        "start": 768,
                        "end": 772
                      }
                    },
                    "loc": {
                      "start": 768,
                      "end": 772
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
                            "start": 786,
                            "end": 794
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 783,
                          "end": 794
                        }
                      }
                    ],
                    "loc": {
                      "start": 773,
                      "end": 800
                    }
                  },
                  "loc": {
                    "start": 761,
                    "end": 800
                  }
                }
              ],
              "loc": {
                "start": 695,
                "end": 802
              }
            },
            "loc": {
              "start": 689,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 803,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 803,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 815,
                "end": 829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 815,
              "end": 829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 830,
                "end": 835
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 830,
              "end": 835
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 836,
                "end": 845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 836,
              "end": 845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 846,
                "end": 850
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
                      "start": 860,
                      "end": 868
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 857,
                    "end": 868
                  }
                }
              ],
              "loc": {
                "start": 851,
                "end": 870
              }
            },
            "loc": {
              "start": 846,
              "end": 870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 871,
                "end": 885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 871,
              "end": 885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 886,
                "end": 891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 886,
              "end": 891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 892,
                "end": 895
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
                      "start": 902,
                      "end": 911
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 902,
                    "end": 911
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 916,
                      "end": 927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 916,
                    "end": 927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 932,
                      "end": 943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 932,
                    "end": 943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 948,
                      "end": 957
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 948,
                    "end": 957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 962,
                      "end": 969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 962,
                    "end": 969
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 974,
                      "end": 982
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 974,
                    "end": 982
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 987,
                      "end": 999
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 987,
                    "end": 999
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1004,
                      "end": 1012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1004,
                    "end": 1012
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1017,
                      "end": 1025
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1017,
                    "end": 1025
                  }
                }
              ],
              "loc": {
                "start": 896,
                "end": 1027
              }
            },
            "loc": {
              "start": 892,
              "end": 1027
            }
          }
        ],
        "loc": {
          "start": 248,
          "end": 1029
        }
      },
      "loc": {
        "start": 221,
        "end": 1029
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 1039,
          "end": 1055
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 1059,
            "end": 1071
          }
        },
        "loc": {
          "start": 1059,
          "end": 1071
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
                "start": 1074,
                "end": 1076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1074,
              "end": 1076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1077,
                "end": 1083
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1077,
              "end": 1083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1084,
                "end": 1087
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
                      "start": 1094,
                      "end": 1107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1094,
                    "end": 1107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1112,
                      "end": 1121
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1112,
                    "end": 1121
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1126,
                      "end": 1137
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1126,
                    "end": 1137
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1142,
                      "end": 1151
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1142,
                    "end": 1151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1156,
                      "end": 1165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1156,
                    "end": 1165
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1170,
                      "end": 1177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1170,
                    "end": 1177
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1182,
                      "end": 1194
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1182,
                    "end": 1194
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1199,
                      "end": 1207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1199,
                    "end": 1207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1212,
                      "end": 1226
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
                            "start": 1237,
                            "end": 1239
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1237,
                          "end": 1239
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1248,
                            "end": 1258
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1248,
                          "end": 1258
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1267,
                            "end": 1277
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1267,
                          "end": 1277
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1286,
                            "end": 1293
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1286,
                          "end": 1293
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1302,
                            "end": 1313
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1302,
                          "end": 1313
                        }
                      }
                    ],
                    "loc": {
                      "start": 1227,
                      "end": 1319
                    }
                  },
                  "loc": {
                    "start": 1212,
                    "end": 1319
                  }
                }
              ],
              "loc": {
                "start": 1088,
                "end": 1321
              }
            },
            "loc": {
              "start": 1084,
              "end": 1321
            }
          }
        ],
        "loc": {
          "start": 1072,
          "end": 1323
        }
      },
      "loc": {
        "start": 1030,
        "end": 1323
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 1333,
          "end": 1346
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 1350,
            "end": 1358
          }
        },
        "loc": {
          "start": 1350,
          "end": 1358
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
                "start": 1361,
                "end": 1363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1361,
              "end": 1363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1364,
                "end": 1374
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1364,
              "end": 1374
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1375,
                "end": 1385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1375,
              "end": 1385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1386,
                "end": 1390
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1386,
              "end": 1390
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1391,
                "end": 1402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1391,
              "end": 1402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1403,
                "end": 1410
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1403,
              "end": 1410
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1411,
                "end": 1416
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1411,
              "end": 1416
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1417,
                "end": 1427
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1417,
              "end": 1427
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 1428,
                "end": 1441
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
                      "start": 1448,
                      "end": 1450
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1448,
                    "end": 1450
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1455,
                      "end": 1465
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1455,
                    "end": 1465
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1470,
                      "end": 1480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1470,
                    "end": 1480
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1485,
                      "end": 1489
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1485,
                    "end": 1489
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1494,
                      "end": 1505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1494,
                    "end": 1505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 1510,
                      "end": 1517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1510,
                    "end": 1517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 1522,
                      "end": 1527
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1522,
                    "end": 1527
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1532,
                      "end": 1542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1532,
                    "end": 1542
                  }
                }
              ],
              "loc": {
                "start": 1442,
                "end": 1544
              }
            },
            "loc": {
              "start": 1428,
              "end": 1544
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 1545,
                "end": 1557
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
                    "value": "created_at",
                    "loc": {
                      "start": 1571,
                      "end": 1581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1571,
                    "end": 1581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1586,
                      "end": 1596
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1586,
                    "end": 1596
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 1601,
                      "end": 1610
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
                          "value": "labels",
                          "loc": {
                            "start": 1621,
                            "end": 1627
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
                                  "start": 1642,
                                  "end": 1644
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1642,
                                "end": 1644
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 1657,
                                  "end": 1662
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1657,
                                "end": 1662
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 1675,
                                  "end": 1680
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1675,
                                "end": 1680
                              }
                            }
                          ],
                          "loc": {
                            "start": 1628,
                            "end": 1690
                          }
                        },
                        "loc": {
                          "start": 1621,
                          "end": 1690
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1699,
                            "end": 1707
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
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 1725,
                                  "end": 1740
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1722,
                                "end": 1740
                              }
                            }
                          ],
                          "loc": {
                            "start": 1708,
                            "end": 1750
                          }
                        },
                        "loc": {
                          "start": 1699,
                          "end": 1750
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1759,
                            "end": 1761
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1759,
                          "end": 1761
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1770,
                            "end": 1774
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1770,
                          "end": 1774
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1783,
                            "end": 1794
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1783,
                          "end": 1794
                        }
                      }
                    ],
                    "loc": {
                      "start": 1611,
                      "end": 1800
                    }
                  },
                  "loc": {
                    "start": 1601,
                    "end": 1800
                  }
                }
              ],
              "loc": {
                "start": 1558,
                "end": 1802
              }
            },
            "loc": {
              "start": 1545,
              "end": 1802
            }
          }
        ],
        "loc": {
          "start": 1359,
          "end": 1804
        }
      },
      "loc": {
        "start": 1324,
        "end": 1804
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1814,
          "end": 1827
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1831,
            "end": 1839
          }
        },
        "loc": {
          "start": 1831,
          "end": 1839
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
                "start": 1842,
                "end": 1844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1842,
              "end": 1844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1845,
                "end": 1850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1845,
              "end": 1850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1851,
                "end": 1855
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1851,
              "end": 1855
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1856,
                "end": 1863
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1856,
              "end": 1863
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1864,
                "end": 1876
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
                      "start": 1883,
                      "end": 1885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1883,
                    "end": 1885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1890,
                      "end": 1898
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1890,
                    "end": 1898
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1903,
                      "end": 1914
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1903,
                    "end": 1914
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1919,
                      "end": 1923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1919,
                    "end": 1923
                  }
                }
              ],
              "loc": {
                "start": 1877,
                "end": 1925
              }
            },
            "loc": {
              "start": 1864,
              "end": 1925
            }
          }
        ],
        "loc": {
          "start": 1840,
          "end": 1927
        }
      },
      "loc": {
        "start": 1805,
        "end": 1927
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1937,
          "end": 1952
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1956,
            "end": 1964
          }
        },
        "loc": {
          "start": 1956,
          "end": 1964
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
                "start": 1967,
                "end": 1969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1967,
              "end": 1969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1970,
                "end": 1980
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1970,
              "end": 1980
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1981,
                "end": 1991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1981,
              "end": 1991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1992,
                "end": 2001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1992,
              "end": 2001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2002,
                "end": 2009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2002,
              "end": 2009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2010,
                "end": 2018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2010,
              "end": 2018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2019,
                "end": 2029
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
                      "start": 2036,
                      "end": 2038
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2036,
                    "end": 2038
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2043,
                      "end": 2060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2043,
                    "end": 2060
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2065,
                      "end": 2077
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2065,
                    "end": 2077
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2082,
                      "end": 2092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2082,
                    "end": 2092
                  }
                }
              ],
              "loc": {
                "start": 2030,
                "end": 2094
              }
            },
            "loc": {
              "start": 2019,
              "end": 2094
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2095,
                "end": 2106
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
                      "start": 2113,
                      "end": 2115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2113,
                    "end": 2115
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2120,
                      "end": 2134
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2120,
                    "end": 2134
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2139,
                      "end": 2147
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2139,
                    "end": 2147
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2152,
                      "end": 2161
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2152,
                    "end": 2161
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2166,
                      "end": 2176
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2166,
                    "end": 2176
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2181,
                      "end": 2186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2181,
                    "end": 2186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2191,
                      "end": 2198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2191,
                    "end": 2198
                  }
                }
              ],
              "loc": {
                "start": 2107,
                "end": 2200
              }
            },
            "loc": {
              "start": 2095,
              "end": 2200
            }
          }
        ],
        "loc": {
          "start": 1965,
          "end": 2202
        }
      },
      "loc": {
        "start": 1928,
        "end": 2202
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 2212,
          "end": 2225
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2229,
            "end": 2237
          }
        },
        "loc": {
          "start": 2229,
          "end": 2237
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
              "value": "labels",
              "loc": {
                "start": 2240,
                "end": 2246
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
                      "start": 2256,
                      "end": 2266
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2253,
                    "end": 2266
                  }
                }
              ],
              "loc": {
                "start": 2247,
                "end": 2268
              }
            },
            "loc": {
              "start": 2240,
              "end": 2268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 2269,
                "end": 2279
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
                    "value": "labels",
                    "loc": {
                      "start": 2286,
                      "end": 2292
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
                            "start": 2303,
                            "end": 2305
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2303,
                          "end": 2305
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 2314,
                            "end": 2319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2314,
                          "end": 2319
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2328,
                            "end": 2333
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2328,
                          "end": 2333
                        }
                      }
                    ],
                    "loc": {
                      "start": 2293,
                      "end": 2339
                    }
                  },
                  "loc": {
                    "start": 2286,
                    "end": 2339
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2344,
                      "end": 2346
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2344,
                    "end": 2346
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2351,
                      "end": 2355
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2351,
                    "end": 2355
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2360,
                      "end": 2371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2360,
                    "end": 2371
                  }
                }
              ],
              "loc": {
                "start": 2280,
                "end": 2373
              }
            },
            "loc": {
              "start": 2269,
              "end": 2373
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 2374,
                "end": 2382
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
                    "value": "labels",
                    "loc": {
                      "start": 2389,
                      "end": 2395
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
                            "start": 2409,
                            "end": 2419
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2406,
                          "end": 2419
                        }
                      }
                    ],
                    "loc": {
                      "start": 2396,
                      "end": 2425
                    }
                  },
                  "loc": {
                    "start": 2389,
                    "end": 2425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2430,
                      "end": 2442
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
                            "start": 2453,
                            "end": 2455
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2453,
                          "end": 2455
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2464,
                            "end": 2472
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2464,
                          "end": 2472
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2481,
                            "end": 2492
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2481,
                          "end": 2492
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 2501,
                            "end": 2505
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2501,
                          "end": 2505
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2514,
                            "end": 2518
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2514,
                          "end": 2518
                        }
                      }
                    ],
                    "loc": {
                      "start": 2443,
                      "end": 2524
                    }
                  },
                  "loc": {
                    "start": 2430,
                    "end": 2524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2529,
                      "end": 2531
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2529,
                    "end": 2531
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 2536,
                      "end": 2558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2536,
                    "end": 2558
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnOrganizationProfile",
                    "loc": {
                      "start": 2563,
                      "end": 2588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2563,
                    "end": 2588
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 2593,
                      "end": 2605
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
                            "start": 2616,
                            "end": 2618
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2616,
                          "end": 2618
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 2627,
                            "end": 2633
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2627,
                          "end": 2633
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 2642,
                            "end": 2645
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
                                  "start": 2660,
                                  "end": 2673
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2660,
                                "end": 2673
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 2686,
                                  "end": 2695
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2686,
                                "end": 2695
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 2708,
                                  "end": 2719
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2708,
                                "end": 2719
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 2732,
                                  "end": 2741
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2732,
                                "end": 2741
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 2754,
                                  "end": 2763
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2754,
                                "end": 2763
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 2776,
                                  "end": 2783
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2776,
                                "end": 2783
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 2796,
                                  "end": 2808
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2796,
                                "end": 2808
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 2821,
                                  "end": 2829
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2821,
                                "end": 2829
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 2842,
                                  "end": 2856
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
                                        "start": 2875,
                                        "end": 2877
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2875,
                                      "end": 2877
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2894,
                                        "end": 2904
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2894,
                                      "end": 2904
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 2921,
                                        "end": 2931
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2921,
                                      "end": 2931
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 2948,
                                        "end": 2955
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2948,
                                      "end": 2955
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 2972,
                                        "end": 2983
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2972,
                                      "end": 2983
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2857,
                                  "end": 2997
                                }
                              },
                              "loc": {
                                "start": 2842,
                                "end": 2997
                              }
                            }
                          ],
                          "loc": {
                            "start": 2646,
                            "end": 3007
                          }
                        },
                        "loc": {
                          "start": 2642,
                          "end": 3007
                        }
                      }
                    ],
                    "loc": {
                      "start": 2606,
                      "end": 3013
                    }
                  },
                  "loc": {
                    "start": 2593,
                    "end": 3013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 3018,
                      "end": 3035
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
                          "value": "members",
                          "loc": {
                            "start": 3046,
                            "end": 3053
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
                                  "start": 3068,
                                  "end": 3070
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3068,
                                "end": 3070
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3083,
                                  "end": 3093
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3083,
                                "end": 3093
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3106,
                                  "end": 3116
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3106,
                                "end": 3116
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3129,
                                  "end": 3136
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3129,
                                "end": 3136
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3149,
                                  "end": 3160
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3149,
                                "end": 3160
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 3173,
                                  "end": 3178
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
                                        "start": 3197,
                                        "end": 3199
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3197,
                                      "end": 3199
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3216,
                                        "end": 3226
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3216,
                                      "end": 3226
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3243,
                                        "end": 3253
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3243,
                                      "end": 3253
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3270,
                                        "end": 3274
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3270,
                                      "end": 3274
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3291,
                                        "end": 3302
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3291,
                                      "end": 3302
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 3319,
                                        "end": 3331
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3319,
                                      "end": 3331
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 3348,
                                        "end": 3360
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
                                              "start": 3383,
                                              "end": 3385
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3383,
                                            "end": 3385
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 3406,
                                              "end": 3412
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3406,
                                            "end": 3412
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 3433,
                                              "end": 3436
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
                                                    "start": 3463,
                                                    "end": 3476
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3463,
                                                  "end": 3476
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 3501,
                                                    "end": 3510
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3501,
                                                  "end": 3510
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 3535,
                                                    "end": 3546
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3535,
                                                  "end": 3546
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 3571,
                                                    "end": 3580
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3571,
                                                  "end": 3580
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 3605,
                                                    "end": 3614
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3605,
                                                  "end": 3614
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 3639,
                                                    "end": 3646
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3639,
                                                  "end": 3646
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 3671,
                                                    "end": 3683
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3671,
                                                  "end": 3683
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 3708,
                                                    "end": 3716
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3708,
                                                  "end": 3716
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 3741,
                                                    "end": 3755
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
                                                          "start": 3786,
                                                          "end": 3788
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3786,
                                                        "end": 3788
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 3817,
                                                          "end": 3827
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3817,
                                                        "end": 3827
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 3856,
                                                          "end": 3866
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3856,
                                                        "end": 3866
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 3895,
                                                          "end": 3902
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3895,
                                                        "end": 3902
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 3931,
                                                          "end": 3942
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3931,
                                                        "end": 3942
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 3756,
                                                    "end": 3968
                                                  }
                                                },
                                                "loc": {
                                                  "start": 3741,
                                                  "end": 3968
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3437,
                                              "end": 3990
                                            }
                                          },
                                          "loc": {
                                            "start": 3433,
                                            "end": 3990
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3361,
                                        "end": 4008
                                      }
                                    },
                                    "loc": {
                                      "start": 3348,
                                      "end": 4008
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4025,
                                        "end": 4037
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
                                              "start": 4060,
                                              "end": 4062
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4060,
                                            "end": 4062
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4083,
                                              "end": 4091
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4083,
                                            "end": 4091
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4112,
                                              "end": 4123
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4112,
                                            "end": 4123
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4038,
                                        "end": 4141
                                      }
                                    },
                                    "loc": {
                                      "start": 4025,
                                      "end": 4141
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3179,
                                  "end": 4155
                                }
                              },
                              "loc": {
                                "start": 3173,
                                "end": 4155
                              }
                            }
                          ],
                          "loc": {
                            "start": 3054,
                            "end": 4165
                          }
                        },
                        "loc": {
                          "start": 3046,
                          "end": 4165
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4174,
                            "end": 4176
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4174,
                          "end": 4176
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4185,
                            "end": 4195
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4185,
                          "end": 4195
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4204,
                            "end": 4214
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4204,
                          "end": 4214
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4223,
                            "end": 4227
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4223,
                          "end": 4227
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 4236,
                            "end": 4247
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4236,
                          "end": 4247
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 4256,
                            "end": 4268
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4256,
                          "end": 4268
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 4277,
                            "end": 4289
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
                                  "start": 4304,
                                  "end": 4306
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4304,
                                "end": 4306
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 4319,
                                  "end": 4325
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4319,
                                "end": 4325
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4338,
                                  "end": 4341
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
                                        "start": 4360,
                                        "end": 4373
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4360,
                                      "end": 4373
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 4390,
                                        "end": 4399
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4390,
                                      "end": 4399
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 4416,
                                        "end": 4427
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4416,
                                      "end": 4427
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 4444,
                                        "end": 4453
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4444,
                                      "end": 4453
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 4470,
                                        "end": 4479
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4470,
                                      "end": 4479
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 4496,
                                        "end": 4503
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4496,
                                      "end": 4503
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 4520,
                                        "end": 4532
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4520,
                                      "end": 4532
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 4549,
                                        "end": 4557
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4549,
                                      "end": 4557
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 4574,
                                        "end": 4588
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
                                              "start": 4611,
                                              "end": 4613
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4611,
                                            "end": 4613
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4634,
                                              "end": 4644
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4634,
                                            "end": 4644
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4665,
                                              "end": 4675
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4665,
                                            "end": 4675
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 4696,
                                              "end": 4703
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4696,
                                            "end": 4703
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 4724,
                                              "end": 4735
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4724,
                                            "end": 4735
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4589,
                                        "end": 4753
                                      }
                                    },
                                    "loc": {
                                      "start": 4574,
                                      "end": 4753
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4342,
                                  "end": 4767
                                }
                              },
                              "loc": {
                                "start": 4338,
                                "end": 4767
                              }
                            }
                          ],
                          "loc": {
                            "start": 4290,
                            "end": 4777
                          }
                        },
                        "loc": {
                          "start": 4277,
                          "end": 4777
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 4786,
                            "end": 4798
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
                                  "start": 4813,
                                  "end": 4815
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4813,
                                "end": 4815
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
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
                                "value": "description",
                                "loc": {
                                  "start": 4849,
                                  "end": 4860
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4849,
                                "end": 4860
                              }
                            }
                          ],
                          "loc": {
                            "start": 4799,
                            "end": 4870
                          }
                        },
                        "loc": {
                          "start": 4786,
                          "end": 4870
                        }
                      }
                    ],
                    "loc": {
                      "start": 3036,
                      "end": 4876
                    }
                  },
                  "loc": {
                    "start": 3018,
                    "end": 4876
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 4881,
                      "end": 4895
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4881,
                    "end": 4895
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 4900,
                      "end": 4912
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4900,
                    "end": 4912
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 4917,
                      "end": 4920
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
                            "start": 4931,
                            "end": 4940
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4931,
                          "end": 4940
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 4949,
                            "end": 4958
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4949,
                          "end": 4958
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 4967,
                            "end": 4976
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4967,
                          "end": 4976
                        }
                      }
                    ],
                    "loc": {
                      "start": 4921,
                      "end": 4982
                    }
                  },
                  "loc": {
                    "start": 4917,
                    "end": 4982
                  }
                }
              ],
              "loc": {
                "start": 2383,
                "end": 4984
              }
            },
            "loc": {
              "start": 2374,
              "end": 4984
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 4985,
                "end": 4996
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
                    "value": "projectVersion",
                    "loc": {
                      "start": 5003,
                      "end": 5017
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
                            "start": 5028,
                            "end": 5030
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5028,
                          "end": 5030
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5039,
                            "end": 5049
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5039,
                          "end": 5049
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5058,
                            "end": 5066
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5058,
                          "end": 5066
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5075,
                            "end": 5084
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5075,
                          "end": 5084
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 5093,
                            "end": 5105
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5093,
                          "end": 5105
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 5114,
                            "end": 5126
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5114,
                          "end": 5126
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 5135,
                            "end": 5139
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
                                  "start": 5154,
                                  "end": 5156
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5154,
                                "end": 5156
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 5169,
                                  "end": 5178
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5169,
                                "end": 5178
                              }
                            }
                          ],
                          "loc": {
                            "start": 5140,
                            "end": 5188
                          }
                        },
                        "loc": {
                          "start": 5135,
                          "end": 5188
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5197,
                            "end": 5209
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
                                  "start": 5224,
                                  "end": 5226
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5224,
                                "end": 5226
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5239,
                                  "end": 5247
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5239,
                                "end": 5247
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5260,
                                  "end": 5271
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5260,
                                "end": 5271
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 5284,
                                  "end": 5288
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5284,
                                "end": 5288
                              }
                            }
                          ],
                          "loc": {
                            "start": 5210,
                            "end": 5298
                          }
                        },
                        "loc": {
                          "start": 5197,
                          "end": 5298
                        }
                      }
                    ],
                    "loc": {
                      "start": 5018,
                      "end": 5304
                    }
                  },
                  "loc": {
                    "start": 5003,
                    "end": 5304
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5309,
                      "end": 5311
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5309,
                    "end": 5311
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5316,
                      "end": 5325
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5316,
                    "end": 5325
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 5330,
                      "end": 5349
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5330,
                    "end": 5349
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 5354,
                      "end": 5369
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5354,
                    "end": 5369
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 5374,
                      "end": 5383
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5374,
                    "end": 5383
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 5388,
                      "end": 5399
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5388,
                    "end": 5399
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 5404,
                      "end": 5415
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5404,
                    "end": 5415
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5420,
                      "end": 5424
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5420,
                    "end": 5424
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 5429,
                      "end": 5435
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5429,
                    "end": 5435
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 5440,
                      "end": 5450
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5440,
                    "end": 5450
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5455,
                      "end": 5467
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5481,
                            "end": 5497
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5478,
                          "end": 5497
                        }
                      }
                    ],
                    "loc": {
                      "start": 5468,
                      "end": 5503
                    }
                  },
                  "loc": {
                    "start": 5455,
                    "end": 5503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 5508,
                      "end": 5512
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
                          "value": "User_nav",
                          "loc": {
                            "start": 5526,
                            "end": 5534
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5523,
                          "end": 5534
                        }
                      }
                    ],
                    "loc": {
                      "start": 5513,
                      "end": 5540
                    }
                  },
                  "loc": {
                    "start": 5508,
                    "end": 5540
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5545,
                      "end": 5548
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
                            "start": 5559,
                            "end": 5568
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5559,
                          "end": 5568
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5577,
                            "end": 5586
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5577,
                          "end": 5586
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5595,
                            "end": 5602
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5595,
                          "end": 5602
                        }
                      }
                    ],
                    "loc": {
                      "start": 5549,
                      "end": 5608
                    }
                  },
                  "loc": {
                    "start": 5545,
                    "end": 5608
                  }
                }
              ],
              "loc": {
                "start": 4997,
                "end": 5610
              }
            },
            "loc": {
              "start": 4985,
              "end": 5610
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 5611,
                "end": 5622
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
                    "value": "routineVersion",
                    "loc": {
                      "start": 5629,
                      "end": 5643
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
                            "start": 5654,
                            "end": 5656
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5654,
                          "end": 5656
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5665,
                            "end": 5675
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5665,
                          "end": 5675
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 5684,
                            "end": 5697
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5684,
                          "end": 5697
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 5706,
                            "end": 5716
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5706,
                          "end": 5716
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 5725,
                            "end": 5734
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5725,
                          "end": 5734
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5743,
                            "end": 5751
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5743,
                          "end": 5751
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5760,
                            "end": 5769
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5760,
                          "end": 5769
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 5778,
                            "end": 5782
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
                                  "start": 5797,
                                  "end": 5799
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5797,
                                "end": 5799
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 5812,
                                  "end": 5822
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5812,
                                "end": 5822
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
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
                            }
                          ],
                          "loc": {
                            "start": 5783,
                            "end": 5854
                          }
                        },
                        "loc": {
                          "start": 5778,
                          "end": 5854
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5863,
                            "end": 5875
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
                                  "start": 5890,
                                  "end": 5892
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5890,
                                "end": 5892
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5905,
                                  "end": 5913
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5905,
                                "end": 5913
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5926,
                                  "end": 5937
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5926,
                                "end": 5937
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 5950,
                                  "end": 5962
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5950,
                                "end": 5962
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 5975,
                                  "end": 5979
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5975,
                                "end": 5979
                              }
                            }
                          ],
                          "loc": {
                            "start": 5876,
                            "end": 5989
                          }
                        },
                        "loc": {
                          "start": 5863,
                          "end": 5989
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 5998,
                            "end": 6010
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5998,
                          "end": 6010
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
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
                      }
                    ],
                    "loc": {
                      "start": 5644,
                      "end": 6037
                    }
                  },
                  "loc": {
                    "start": 5629,
                    "end": 6037
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6042,
                      "end": 6044
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6042,
                    "end": 6044
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6049,
                      "end": 6058
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6049,
                    "end": 6058
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6063,
                      "end": 6082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6063,
                    "end": 6082
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6087,
                      "end": 6102
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6087,
                    "end": 6102
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
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
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6121,
                      "end": 6132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6121,
                    "end": 6132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6137,
                      "end": 6148
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6137,
                    "end": 6148
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6153,
                      "end": 6157
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6153,
                    "end": 6157
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6162,
                      "end": 6168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6162,
                    "end": 6168
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6173,
                      "end": 6183
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6173,
                    "end": 6183
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 6188,
                      "end": 6199
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6188,
                    "end": 6199
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 6204,
                      "end": 6223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6204,
                    "end": 6223
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 6228,
                      "end": 6240
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6254,
                            "end": 6270
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6251,
                          "end": 6270
                        }
                      }
                    ],
                    "loc": {
                      "start": 6241,
                      "end": 6276
                    }
                  },
                  "loc": {
                    "start": 6228,
                    "end": 6276
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6281,
                      "end": 6285
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
                          "value": "User_nav",
                          "loc": {
                            "start": 6299,
                            "end": 6307
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6296,
                          "end": 6307
                        }
                      }
                    ],
                    "loc": {
                      "start": 6286,
                      "end": 6313
                    }
                  },
                  "loc": {
                    "start": 6281,
                    "end": 6313
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6318,
                      "end": 6321
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
                            "start": 6332,
                            "end": 6341
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6332,
                          "end": 6341
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6350,
                            "end": 6359
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6350,
                          "end": 6359
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6368,
                            "end": 6375
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6368,
                          "end": 6375
                        }
                      }
                    ],
                    "loc": {
                      "start": 6322,
                      "end": 6381
                    }
                  },
                  "loc": {
                    "start": 6318,
                    "end": 6381
                  }
                }
              ],
              "loc": {
                "start": 5623,
                "end": 6383
              }
            },
            "loc": {
              "start": 5611,
              "end": 6383
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6384,
                "end": 6386
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6384,
              "end": 6386
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6387,
                "end": 6397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6387,
              "end": 6397
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6398,
                "end": 6408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6398,
              "end": 6408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 6409,
                "end": 6418
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6409,
              "end": 6418
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 6419,
                "end": 6426
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6419,
              "end": 6426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 6427,
                "end": 6435
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6427,
              "end": 6435
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 6436,
                "end": 6446
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
                      "start": 6453,
                      "end": 6455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6453,
                    "end": 6455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 6460,
                      "end": 6477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6460,
                    "end": 6477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 6482,
                      "end": 6494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6482,
                    "end": 6494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 6499,
                      "end": 6509
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6499,
                    "end": 6509
                  }
                }
              ],
              "loc": {
                "start": 6447,
                "end": 6511
              }
            },
            "loc": {
              "start": 6436,
              "end": 6511
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 6512,
                "end": 6523
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
                      "start": 6530,
                      "end": 6532
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6530,
                    "end": 6532
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 6537,
                      "end": 6551
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6537,
                    "end": 6551
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 6556,
                      "end": 6564
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6556,
                    "end": 6564
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 6569,
                      "end": 6578
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6569,
                    "end": 6578
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 6583,
                      "end": 6593
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6583,
                    "end": 6593
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 6598,
                      "end": 6603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6598,
                    "end": 6603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 6608,
                      "end": 6615
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6608,
                    "end": 6615
                  }
                }
              ],
              "loc": {
                "start": 6524,
                "end": 6617
              }
            },
            "loc": {
              "start": 6512,
              "end": 6617
            }
          }
        ],
        "loc": {
          "start": 2238,
          "end": 6619
        }
      },
      "loc": {
        "start": 2203,
        "end": 6619
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 6629,
          "end": 6637
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 6641,
            "end": 6644
          }
        },
        "loc": {
          "start": 6641,
          "end": 6644
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
                "start": 6647,
                "end": 6649
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6647,
              "end": 6649
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6650,
                "end": 6660
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6650,
              "end": 6660
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 6661,
                "end": 6664
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6661,
              "end": 6664
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6665,
                "end": 6674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6665,
              "end": 6674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6675,
                "end": 6687
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
                      "start": 6694,
                      "end": 6696
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6694,
                    "end": 6696
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6701,
                      "end": 6709
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6701,
                    "end": 6709
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6714,
                      "end": 6725
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6714,
                    "end": 6725
                  }
                }
              ],
              "loc": {
                "start": 6688,
                "end": 6727
              }
            },
            "loc": {
              "start": 6675,
              "end": 6727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6728,
                "end": 6731
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
                      "start": 6738,
                      "end": 6743
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6738,
                    "end": 6743
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6748,
                      "end": 6760
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6748,
                    "end": 6760
                  }
                }
              ],
              "loc": {
                "start": 6732,
                "end": 6762
              }
            },
            "loc": {
              "start": 6728,
              "end": 6762
            }
          }
        ],
        "loc": {
          "start": 6645,
          "end": 6764
        }
      },
      "loc": {
        "start": 6620,
        "end": 6764
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 6774,
          "end": 6782
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 6786,
            "end": 6790
          }
        },
        "loc": {
          "start": 6786,
          "end": 6790
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
                "start": 6793,
                "end": 6795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6793,
              "end": 6795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 6796,
                "end": 6801
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6796,
              "end": 6801
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6802,
                "end": 6806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6802,
              "end": 6806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 6807,
                "end": 6813
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6807,
              "end": 6813
            }
          }
        ],
        "loc": {
          "start": 6791,
          "end": 6815
        }
      },
      "loc": {
        "start": 6765,
        "end": 6815
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "home",
      "loc": {
        "start": 6823,
        "end": 6827
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
              "start": 6829,
              "end": 6834
            }
          },
          "loc": {
            "start": 6828,
            "end": 6834
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "HomeInput",
              "loc": {
                "start": 6836,
                "end": 6845
              }
            },
            "loc": {
              "start": 6836,
              "end": 6845
            }
          },
          "loc": {
            "start": 6836,
            "end": 6846
          }
        },
        "directives": [],
        "loc": {
          "start": 6828,
          "end": 6846
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
            "value": "home",
            "loc": {
              "start": 6852,
              "end": 6856
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 6857,
                  "end": 6862
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 6865,
                    "end": 6870
                  }
                },
                "loc": {
                  "start": 6864,
                  "end": 6870
                }
              },
              "loc": {
                "start": 6857,
                "end": 6870
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
                  "value": "notes",
                  "loc": {
                    "start": 6878,
                    "end": 6883
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
                          "start": 6897,
                          "end": 6906
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 6894,
                        "end": 6906
                      }
                    }
                  ],
                  "loc": {
                    "start": 6884,
                    "end": 6912
                  }
                },
                "loc": {
                  "start": 6878,
                  "end": 6912
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 6917,
                    "end": 6926
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
                        "value": "Reminder_full",
                        "loc": {
                          "start": 6940,
                          "end": 6953
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 6937,
                        "end": 6953
                      }
                    }
                  ],
                  "loc": {
                    "start": 6927,
                    "end": 6959
                  }
                },
                "loc": {
                  "start": 6917,
                  "end": 6959
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 6964,
                    "end": 6973
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
                        "value": "Resource_list",
                        "loc": {
                          "start": 6987,
                          "end": 7000
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 6984,
                        "end": 7000
                      }
                    }
                  ],
                  "loc": {
                    "start": 6974,
                    "end": 7006
                  }
                },
                "loc": {
                  "start": 6964,
                  "end": 7006
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 7011,
                    "end": 7020
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
                        "value": "Schedule_list",
                        "loc": {
                          "start": 7034,
                          "end": 7047
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7031,
                        "end": 7047
                      }
                    }
                  ],
                  "loc": {
                    "start": 7021,
                    "end": 7053
                  }
                },
                "loc": {
                  "start": 7011,
                  "end": 7053
                }
              }
            ],
            "loc": {
              "start": 6872,
              "end": 7057
            }
          },
          "loc": {
            "start": 6852,
            "end": 7057
          }
        }
      ],
      "loc": {
        "start": 6848,
        "end": 7059
      }
    },
    "loc": {
      "start": 6817,
      "end": 7059
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
