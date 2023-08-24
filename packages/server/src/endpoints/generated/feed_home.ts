export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 7146,
          "end": 7150
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7151,
              "end": 7156
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7159,
                "end": 7164
              }
            },
            "loc": {
              "start": 7158,
              "end": 7164
            }
          },
          "loc": {
            "start": 7151,
            "end": 7164
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
                "start": 7172,
                "end": 7177
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
                      "start": 7191,
                      "end": 7200
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7188,
                    "end": 7200
                  }
                }
              ],
              "loc": {
                "start": 7178,
                "end": 7206
              }
            },
            "loc": {
              "start": 7172,
              "end": 7206
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 7211,
                "end": 7220
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
                      "start": 7234,
                      "end": 7247
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7231,
                    "end": 7247
                  }
                }
              ],
              "loc": {
                "start": 7221,
                "end": 7253
              }
            },
            "loc": {
              "start": 7211,
              "end": 7253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 7258,
                "end": 7267
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
                      "start": 7281,
                      "end": 7294
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7278,
                    "end": 7294
                  }
                }
              ],
              "loc": {
                "start": 7268,
                "end": 7300
              }
            },
            "loc": {
              "start": 7258,
              "end": 7300
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 7305,
                "end": 7314
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
                      "start": 7328,
                      "end": 7341
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7325,
                    "end": 7341
                  }
                }
              ],
              "loc": {
                "start": 7315,
                "end": 7347
              }
            },
            "loc": {
              "start": 7305,
              "end": 7347
            }
          }
        ],
        "loc": {
          "start": 7166,
          "end": 7351
        }
      },
      "loc": {
        "start": 7146,
        "end": 7351
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
                    "value": "pages",
                    "loc": {
                      "start": 349,
                      "end": 354
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
                            "start": 369,
                            "end": 371
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 369,
                          "end": 371
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 384,
                            "end": 393
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 384,
                          "end": 393
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 406,
                            "end": 410
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 406,
                          "end": 410
                        }
                      }
                    ],
                    "loc": {
                      "start": 355,
                      "end": 420
                    }
                  },
                  "loc": {
                    "start": 349,
                    "end": 420
                  }
                }
              ],
              "loc": {
                "start": 278,
                "end": 426
              }
            },
            "loc": {
              "start": 265,
              "end": 426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 431,
                "end": 433
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 431,
              "end": 433
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 438,
                "end": 448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 438,
              "end": 448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 453,
                "end": 463
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 453,
              "end": 463
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 468,
                "end": 476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 468,
              "end": 476
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 481,
                "end": 490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 481,
              "end": 490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 495,
                "end": 507
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 495,
              "end": 507
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 512,
                "end": 524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 512,
              "end": 524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 529,
                "end": 541
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 529,
              "end": 541
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 546,
                "end": 549
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
                      "start": 560,
                      "end": 570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 560,
                    "end": 570
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 579,
                      "end": 586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 579,
                    "end": 586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 595,
                      "end": 604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 595,
                    "end": 604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 613,
                      "end": 622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 613,
                    "end": 622
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 631,
                      "end": 640
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 631,
                    "end": 640
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 649,
                      "end": 655
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 649,
                    "end": 655
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 664,
                      "end": 671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 664,
                    "end": 671
                  }
                }
              ],
              "loc": {
                "start": 550,
                "end": 677
              }
            },
            "loc": {
              "start": 546,
              "end": 677
            }
          }
        ],
        "loc": {
          "start": 259,
          "end": 679
        }
      },
      "loc": {
        "start": 250,
        "end": 679
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 680,
          "end": 682
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 680,
        "end": 682
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 683,
          "end": 693
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 683,
        "end": 693
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 694,
          "end": 704
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 694,
        "end": 704
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 705,
          "end": 714
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 705,
        "end": 714
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 715,
          "end": 726
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 715,
        "end": 726
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 727,
          "end": 733
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
                "start": 743,
                "end": 753
              }
            },
            "directives": [],
            "loc": {
              "start": 740,
              "end": 753
            }
          }
        ],
        "loc": {
          "start": 734,
          "end": 755
        }
      },
      "loc": {
        "start": 727,
        "end": 755
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 756,
          "end": 761
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
                  "start": 775,
                  "end": 787
                }
              },
              "loc": {
                "start": 775,
                "end": 787
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
                      "start": 801,
                      "end": 817
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 798,
                    "end": 817
                  }
                }
              ],
              "loc": {
                "start": 788,
                "end": 823
              }
            },
            "loc": {
              "start": 768,
              "end": 823
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
                  "start": 835,
                  "end": 839
                }
              },
              "loc": {
                "start": 835,
                "end": 839
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
                      "start": 853,
                      "end": 861
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 850,
                    "end": 861
                  }
                }
              ],
              "loc": {
                "start": 840,
                "end": 867
              }
            },
            "loc": {
              "start": 828,
              "end": 867
            }
          }
        ],
        "loc": {
          "start": 762,
          "end": 869
        }
      },
      "loc": {
        "start": 756,
        "end": 869
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 870,
          "end": 881
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 870,
        "end": 881
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 882,
          "end": 896
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 882,
        "end": 896
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 897,
          "end": 902
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 897,
        "end": 902
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 903,
          "end": 912
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 903,
        "end": 912
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 913,
          "end": 917
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
                "start": 927,
                "end": 935
              }
            },
            "directives": [],
            "loc": {
              "start": 924,
              "end": 935
            }
          }
        ],
        "loc": {
          "start": 918,
          "end": 937
        }
      },
      "loc": {
        "start": 913,
        "end": 937
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 938,
          "end": 952
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 938,
        "end": 952
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 953,
          "end": 958
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 953,
        "end": 958
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 959,
          "end": 962
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
                "start": 969,
                "end": 978
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 969,
              "end": 978
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 983,
                "end": 994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 983,
              "end": 994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 999,
                "end": 1010
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 999,
              "end": 1010
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1015,
                "end": 1024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1015,
              "end": 1024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1029,
                "end": 1036
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1029,
              "end": 1036
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1041,
                "end": 1049
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1041,
              "end": 1049
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1054,
                "end": 1066
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1054,
              "end": 1066
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1071,
                "end": 1079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1071,
              "end": 1079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1084,
                "end": 1092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1084,
              "end": 1092
            }
          }
        ],
        "loc": {
          "start": 963,
          "end": 1094
        }
      },
      "loc": {
        "start": 959,
        "end": 1094
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1141,
          "end": 1143
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1141,
        "end": 1143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1144,
          "end": 1155
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1144,
        "end": 1155
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1156,
          "end": 1162
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1156,
        "end": 1162
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1163,
          "end": 1175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1163,
        "end": 1175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1176,
          "end": 1179
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
                "start": 1186,
                "end": 1199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1204,
                "end": 1213
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1204,
              "end": 1213
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1218,
                "end": 1229
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1218,
              "end": 1229
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1234,
                "end": 1243
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1234,
              "end": 1243
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1248,
                "end": 1257
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1248,
              "end": 1257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1262,
                "end": 1269
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1262,
              "end": 1269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1274,
                "end": 1286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1274,
              "end": 1286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1291,
                "end": 1299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1291,
              "end": 1299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1304,
                "end": 1318
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
                      "start": 1329,
                      "end": 1331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1329,
                    "end": 1331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1340,
                      "end": 1350
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1340,
                    "end": 1350
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1359,
                      "end": 1369
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1369
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1378,
                      "end": 1385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1378,
                    "end": 1385
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1394,
                      "end": 1405
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1394,
                    "end": 1405
                  }
                }
              ],
              "loc": {
                "start": 1319,
                "end": 1411
              }
            },
            "loc": {
              "start": 1304,
              "end": 1411
            }
          }
        ],
        "loc": {
          "start": 1180,
          "end": 1413
        }
      },
      "loc": {
        "start": 1176,
        "end": 1413
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1453,
          "end": 1455
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1453,
        "end": 1455
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1456,
          "end": 1466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1456,
        "end": 1466
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1467,
          "end": 1477
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1467,
        "end": 1477
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1478,
          "end": 1482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1478,
        "end": 1482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 1483,
          "end": 1494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1483,
        "end": 1494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 1495,
          "end": 1502
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1495,
        "end": 1502
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1503,
          "end": 1508
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1503,
        "end": 1508
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 1509,
          "end": 1519
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1509,
        "end": 1519
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 1520,
          "end": 1533
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
                "start": 1540,
                "end": 1542
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1540,
              "end": 1542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1547,
                "end": 1557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1547,
              "end": 1557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1562,
                "end": 1572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1562,
              "end": 1572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1577,
                "end": 1581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1577,
              "end": 1581
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1586,
                "end": 1597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1586,
              "end": 1597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1602,
                "end": 1609
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1602,
              "end": 1609
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1614,
                "end": 1619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1614,
              "end": 1619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1624,
                "end": 1634
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1624,
              "end": 1634
            }
          }
        ],
        "loc": {
          "start": 1534,
          "end": 1636
        }
      },
      "loc": {
        "start": 1520,
        "end": 1636
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 1637,
          "end": 1649
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
                "start": 1656,
                "end": 1658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1656,
              "end": 1658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1663,
                "end": 1673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1663,
              "end": 1673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1678,
                "end": 1688
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1678,
              "end": 1688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 1693,
                "end": 1702
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
                      "start": 1713,
                      "end": 1719
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
                            "start": 1734,
                            "end": 1736
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1734,
                          "end": 1736
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 1749,
                            "end": 1754
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1749,
                          "end": 1754
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 1767,
                            "end": 1772
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1767,
                          "end": 1772
                        }
                      }
                    ],
                    "loc": {
                      "start": 1720,
                      "end": 1782
                    }
                  },
                  "loc": {
                    "start": 1713,
                    "end": 1782
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 1791,
                      "end": 1799
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
                            "start": 1817,
                            "end": 1832
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1814,
                          "end": 1832
                        }
                      }
                    ],
                    "loc": {
                      "start": 1800,
                      "end": 1842
                    }
                  },
                  "loc": {
                    "start": 1791,
                    "end": 1842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1851,
                      "end": 1853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1851,
                    "end": 1853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1862,
                      "end": 1866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1862,
                    "end": 1866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1875,
                      "end": 1886
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1875,
                    "end": 1886
                  }
                }
              ],
              "loc": {
                "start": 1703,
                "end": 1892
              }
            },
            "loc": {
              "start": 1693,
              "end": 1892
            }
          }
        ],
        "loc": {
          "start": 1650,
          "end": 1894
        }
      },
      "loc": {
        "start": 1637,
        "end": 1894
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1934,
          "end": 1936
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1934,
        "end": 1936
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1937,
          "end": 1942
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1937,
        "end": 1942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1943,
          "end": 1947
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1943,
        "end": 1947
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1948,
          "end": 1955
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1948,
        "end": 1955
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1956,
          "end": 1968
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
                "start": 1975,
                "end": 1977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1975,
              "end": 1977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1982,
                "end": 1990
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1982,
              "end": 1990
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1995,
                "end": 2006
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1995,
              "end": 2006
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2011,
                "end": 2015
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2011,
              "end": 2015
            }
          }
        ],
        "loc": {
          "start": 1969,
          "end": 2017
        }
      },
      "loc": {
        "start": 1956,
        "end": 2017
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2059,
          "end": 2061
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2059,
        "end": 2061
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2062,
          "end": 2072
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2062,
        "end": 2072
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2073,
          "end": 2083
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2073,
        "end": 2083
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 2084,
          "end": 2093
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2084,
        "end": 2093
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 2094,
          "end": 2101
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2094,
        "end": 2101
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 2102,
          "end": 2110
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2102,
        "end": 2110
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 2111,
          "end": 2121
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
                "start": 2128,
                "end": 2130
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2128,
              "end": 2130
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 2135,
                "end": 2152
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2135,
              "end": 2152
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 2157,
                "end": 2169
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2157,
              "end": 2169
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 2174,
                "end": 2184
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2174,
              "end": 2184
            }
          }
        ],
        "loc": {
          "start": 2122,
          "end": 2186
        }
      },
      "loc": {
        "start": 2111,
        "end": 2186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 2187,
          "end": 2198
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
                "start": 2205,
                "end": 2207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2205,
              "end": 2207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 2212,
                "end": 2226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2212,
              "end": 2226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 2231,
                "end": 2239
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2231,
              "end": 2239
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 2244,
                "end": 2253
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2244,
              "end": 2253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 2258,
                "end": 2268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2258,
              "end": 2268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 2273,
                "end": 2278
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2273,
              "end": 2278
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 2283,
                "end": 2290
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2283,
              "end": 2290
            }
          }
        ],
        "loc": {
          "start": 2199,
          "end": 2292
        }
      },
      "loc": {
        "start": 2187,
        "end": 2292
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2332,
          "end": 2338
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
                "start": 2348,
                "end": 2358
              }
            },
            "directives": [],
            "loc": {
              "start": 2345,
              "end": 2358
            }
          }
        ],
        "loc": {
          "start": 2339,
          "end": 2360
        }
      },
      "loc": {
        "start": 2332,
        "end": 2360
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 2361,
          "end": 2371
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
                "start": 2378,
                "end": 2384
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
                      "start": 2395,
                      "end": 2397
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2395,
                    "end": 2397
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 2406,
                      "end": 2411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2406,
                    "end": 2411
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2420,
                      "end": 2425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2420,
                    "end": 2425
                  }
                }
              ],
              "loc": {
                "start": 2385,
                "end": 2431
              }
            },
            "loc": {
              "start": 2378,
              "end": 2431
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2436,
                "end": 2438
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2436,
              "end": 2438
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2443,
                "end": 2447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2443,
              "end": 2447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2452,
                "end": 2463
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2452,
              "end": 2463
            }
          }
        ],
        "loc": {
          "start": 2372,
          "end": 2465
        }
      },
      "loc": {
        "start": 2361,
        "end": 2465
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 2466,
          "end": 2474
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
                "start": 2481,
                "end": 2487
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
                      "start": 2501,
                      "end": 2511
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2498,
                    "end": 2511
                  }
                }
              ],
              "loc": {
                "start": 2488,
                "end": 2517
              }
            },
            "loc": {
              "start": 2481,
              "end": 2517
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2522,
                "end": 2534
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
                      "start": 2545,
                      "end": 2547
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2545,
                    "end": 2547
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
                    "value": "description",
                    "loc": {
                      "start": 2573,
                      "end": 2584
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2573,
                    "end": 2584
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 2593,
                      "end": 2597
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2593,
                    "end": 2597
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2606,
                      "end": 2610
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2606,
                    "end": 2610
                  }
                }
              ],
              "loc": {
                "start": 2535,
                "end": 2616
              }
            },
            "loc": {
              "start": 2522,
              "end": 2616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2621,
                "end": 2623
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2621,
              "end": 2623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 2628,
                "end": 2650
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2628,
              "end": 2650
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnOrganizationProfile",
              "loc": {
                "start": 2655,
                "end": 2680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2655,
              "end": 2680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 2685,
                "end": 2697
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
                      "start": 2708,
                      "end": 2710
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2708,
                    "end": 2710
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 2719,
                      "end": 2730
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2719,
                    "end": 2730
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 2739,
                      "end": 2745
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2739,
                    "end": 2745
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 2754,
                      "end": 2766
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2754,
                    "end": 2766
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 2775,
                      "end": 2778
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
                            "start": 2793,
                            "end": 2806
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2793,
                          "end": 2806
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 2819,
                            "end": 2828
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2819,
                          "end": 2828
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 2841,
                            "end": 2852
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2841,
                          "end": 2852
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 2865,
                            "end": 2874
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2865,
                          "end": 2874
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 2887,
                            "end": 2896
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2887,
                          "end": 2896
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 2909,
                            "end": 2916
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2909,
                          "end": 2916
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 2929,
                            "end": 2941
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2929,
                          "end": 2941
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 2954,
                            "end": 2962
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2954,
                          "end": 2962
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 2975,
                            "end": 2989
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
                                  "start": 3008,
                                  "end": 3010
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3008,
                                "end": 3010
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3027,
                                  "end": 3037
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3027,
                                "end": 3037
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3054,
                                  "end": 3064
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3054,
                                "end": 3064
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3081,
                                  "end": 3088
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3081,
                                "end": 3088
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3105,
                                  "end": 3116
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3105,
                                "end": 3116
                              }
                            }
                          ],
                          "loc": {
                            "start": 2990,
                            "end": 3130
                          }
                        },
                        "loc": {
                          "start": 2975,
                          "end": 3130
                        }
                      }
                    ],
                    "loc": {
                      "start": 2779,
                      "end": 3140
                    }
                  },
                  "loc": {
                    "start": 2775,
                    "end": 3140
                  }
                }
              ],
              "loc": {
                "start": 2698,
                "end": 3146
              }
            },
            "loc": {
              "start": 2685,
              "end": 3146
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 3151,
                "end": 3168
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
                      "start": 3179,
                      "end": 3186
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
                            "start": 3201,
                            "end": 3203
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3201,
                          "end": 3203
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
                            "start": 3239,
                            "end": 3249
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3239,
                          "end": 3249
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3262,
                            "end": 3269
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3262,
                          "end": 3269
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3282,
                            "end": 3293
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3282,
                          "end": 3293
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 3306,
                            "end": 3311
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
                                  "start": 3330,
                                  "end": 3332
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3330,
                                "end": 3332
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3349,
                                  "end": 3359
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3349,
                                "end": 3359
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3376,
                                  "end": 3386
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3376,
                                "end": 3386
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3403,
                                  "end": 3407
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3403,
                                "end": 3407
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3424,
                                  "end": 3435
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3424,
                                "end": 3435
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 3452,
                                  "end": 3464
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3452,
                                "end": 3464
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 3481,
                                  "end": 3493
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
                                        "start": 3516,
                                        "end": 3518
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3516,
                                      "end": 3518
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 3539,
                                        "end": 3550
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3539,
                                      "end": 3550
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 3571,
                                        "end": 3577
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3571,
                                      "end": 3577
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 3598,
                                        "end": 3610
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3598,
                                      "end": 3610
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3631,
                                        "end": 3634
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
                                              "start": 3661,
                                              "end": 3674
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3661,
                                            "end": 3674
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 3699,
                                              "end": 3708
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3699,
                                            "end": 3708
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 3733,
                                              "end": 3744
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3733,
                                            "end": 3744
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 3769,
                                              "end": 3778
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3769,
                                            "end": 3778
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 3803,
                                              "end": 3812
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3803,
                                            "end": 3812
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 3837,
                                              "end": 3844
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3837,
                                            "end": 3844
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3869,
                                              "end": 3881
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3869,
                                            "end": 3881
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 3906,
                                              "end": 3914
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3906,
                                            "end": 3914
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 3939,
                                              "end": 3953
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
                                                    "start": 3984,
                                                    "end": 3986
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3984,
                                                  "end": 3986
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 4015,
                                                    "end": 4025
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4015,
                                                  "end": 4025
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 4054,
                                                    "end": 4064
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4054,
                                                  "end": 4064
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 4093,
                                                    "end": 4100
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4093,
                                                  "end": 4100
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 4129,
                                                    "end": 4140
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4129,
                                                  "end": 4140
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3954,
                                              "end": 4166
                                            }
                                          },
                                          "loc": {
                                            "start": 3939,
                                            "end": 4166
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3635,
                                        "end": 4188
                                      }
                                    },
                                    "loc": {
                                      "start": 3631,
                                      "end": 4188
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3494,
                                  "end": 4206
                                }
                              },
                              "loc": {
                                "start": 3481,
                                "end": 4206
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4223,
                                  "end": 4235
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
                                        "start": 4258,
                                        "end": 4260
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4258,
                                      "end": 4260
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4281,
                                        "end": 4289
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4281,
                                      "end": 4289
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4310,
                                        "end": 4321
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4310,
                                      "end": 4321
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4236,
                                  "end": 4339
                                }
                              },
                              "loc": {
                                "start": 4223,
                                "end": 4339
                              }
                            }
                          ],
                          "loc": {
                            "start": 3312,
                            "end": 4353
                          }
                        },
                        "loc": {
                          "start": 3306,
                          "end": 4353
                        }
                      }
                    ],
                    "loc": {
                      "start": 3187,
                      "end": 4363
                    }
                  },
                  "loc": {
                    "start": 3179,
                    "end": 4363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4372,
                      "end": 4374
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4372,
                    "end": 4374
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4383,
                      "end": 4393
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4383,
                    "end": 4393
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4402,
                      "end": 4412
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4402,
                    "end": 4412
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4421,
                      "end": 4425
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4421,
                    "end": 4425
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 4434,
                      "end": 4445
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4434,
                    "end": 4445
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 4454,
                      "end": 4466
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4454,
                    "end": 4466
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 4475,
                      "end": 4487
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
                            "start": 4502,
                            "end": 4504
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4502,
                          "end": 4504
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 4517,
                            "end": 4528
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4517,
                          "end": 4528
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 4541,
                            "end": 4547
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4541,
                          "end": 4547
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 4560,
                            "end": 4572
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4560,
                          "end": 4572
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4585,
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
                                "value": "canAddMembers",
                                "loc": {
                                  "start": 4607,
                                  "end": 4620
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4607,
                                "end": 4620
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 4637,
                                  "end": 4646
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4637,
                                "end": 4646
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 4663,
                                  "end": 4674
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4663,
                                "end": 4674
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 4691,
                                  "end": 4700
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4691,
                                "end": 4700
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4717,
                                  "end": 4726
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4717,
                                "end": 4726
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 4743,
                                  "end": 4750
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4743,
                                "end": 4750
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 4767,
                                  "end": 4779
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4767,
                                "end": 4779
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 4796,
                                  "end": 4804
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4796,
                                "end": 4804
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 4821,
                                  "end": 4835
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
                                        "start": 4858,
                                        "end": 4860
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4858,
                                      "end": 4860
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4881,
                                        "end": 4891
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4881,
                                      "end": 4891
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4912,
                                        "end": 4922
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4912,
                                      "end": 4922
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 4943,
                                        "end": 4950
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4943,
                                      "end": 4950
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4971,
                                        "end": 4982
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4971,
                                      "end": 4982
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4836,
                                  "end": 5000
                                }
                              },
                              "loc": {
                                "start": 4821,
                                "end": 5000
                              }
                            }
                          ],
                          "loc": {
                            "start": 4589,
                            "end": 5014
                          }
                        },
                        "loc": {
                          "start": 4585,
                          "end": 5014
                        }
                      }
                    ],
                    "loc": {
                      "start": 4488,
                      "end": 5024
                    }
                  },
                  "loc": {
                    "start": 4475,
                    "end": 5024
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5033,
                      "end": 5045
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
                            "start": 5060,
                            "end": 5062
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5060,
                          "end": 5062
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5075,
                            "end": 5083
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5075,
                          "end": 5083
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5096,
                            "end": 5107
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5096,
                          "end": 5107
                        }
                      }
                    ],
                    "loc": {
                      "start": 5046,
                      "end": 5117
                    }
                  },
                  "loc": {
                    "start": 5033,
                    "end": 5117
                  }
                }
              ],
              "loc": {
                "start": 3169,
                "end": 5123
              }
            },
            "loc": {
              "start": 3151,
              "end": 5123
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 5128,
                "end": 5142
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5128,
              "end": 5142
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 5147,
                "end": 5159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5147,
              "end": 5159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5164,
                "end": 5167
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
                      "start": 5178,
                      "end": 5187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5178,
                    "end": 5187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 5196,
                      "end": 5205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5196,
                    "end": 5205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5214,
                      "end": 5223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5214,
                    "end": 5223
                  }
                }
              ],
              "loc": {
                "start": 5168,
                "end": 5229
              }
            },
            "loc": {
              "start": 5164,
              "end": 5229
            }
          }
        ],
        "loc": {
          "start": 2475,
          "end": 5231
        }
      },
      "loc": {
        "start": 2466,
        "end": 5231
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 5232,
          "end": 5243
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
                "start": 5250,
                "end": 5264
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
                      "start": 5275,
                      "end": 5277
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5275,
                    "end": 5277
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5286,
                      "end": 5296
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5286,
                    "end": 5296
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5305,
                      "end": 5313
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5305,
                    "end": 5313
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5322,
                      "end": 5331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5322,
                    "end": 5331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5340,
                      "end": 5352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5340,
                    "end": 5352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5361,
                      "end": 5373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5361,
                    "end": 5373
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 5382,
                      "end": 5386
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
                            "start": 5401,
                            "end": 5403
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5401,
                          "end": 5403
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5416,
                            "end": 5425
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5416,
                          "end": 5425
                        }
                      }
                    ],
                    "loc": {
                      "start": 5387,
                      "end": 5435
                    }
                  },
                  "loc": {
                    "start": 5382,
                    "end": 5435
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 5444,
                      "end": 5456
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
                            "start": 5471,
                            "end": 5473
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5471,
                          "end": 5473
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5486,
                            "end": 5494
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5486,
                          "end": 5494
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5507,
                            "end": 5518
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5507,
                          "end": 5518
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5531,
                            "end": 5535
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5531,
                          "end": 5535
                        }
                      }
                    ],
                    "loc": {
                      "start": 5457,
                      "end": 5545
                    }
                  },
                  "loc": {
                    "start": 5444,
                    "end": 5545
                  }
                }
              ],
              "loc": {
                "start": 5265,
                "end": 5551
              }
            },
            "loc": {
              "start": 5250,
              "end": 5551
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5556,
                "end": 5558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5556,
              "end": 5558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5563,
                "end": 5572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5563,
              "end": 5572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 5577,
                "end": 5596
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5577,
              "end": 5596
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 5601,
                "end": 5616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5601,
              "end": 5616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 5621,
                "end": 5630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5621,
              "end": 5630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 5635,
                "end": 5646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5635,
              "end": 5646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 5651,
                "end": 5662
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5651,
              "end": 5662
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 5667,
                "end": 5671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5667,
              "end": 5671
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 5676,
                "end": 5682
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5676,
              "end": 5682
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 5687,
                "end": 5697
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5687,
              "end": 5697
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 5702,
                "end": 5714
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
                      "start": 5728,
                      "end": 5744
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5725,
                    "end": 5744
                  }
                }
              ],
              "loc": {
                "start": 5715,
                "end": 5750
              }
            },
            "loc": {
              "start": 5702,
              "end": 5750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 5755,
                "end": 5759
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
                      "start": 5773,
                      "end": 5781
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5770,
                    "end": 5781
                  }
                }
              ],
              "loc": {
                "start": 5760,
                "end": 5787
              }
            },
            "loc": {
              "start": 5755,
              "end": 5787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5792,
                "end": 5795
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
                      "start": 5806,
                      "end": 5815
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5806,
                    "end": 5815
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5824,
                      "end": 5833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5824,
                    "end": 5833
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5842,
                      "end": 5849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5842,
                    "end": 5849
                  }
                }
              ],
              "loc": {
                "start": 5796,
                "end": 5855
              }
            },
            "loc": {
              "start": 5792,
              "end": 5855
            }
          }
        ],
        "loc": {
          "start": 5244,
          "end": 5857
        }
      },
      "loc": {
        "start": 5232,
        "end": 5857
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 5858,
          "end": 5869
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
                "start": 5876,
                "end": 5890
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
                      "start": 5901,
                      "end": 5903
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5901,
                    "end": 5903
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 5912,
                      "end": 5922
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5912,
                    "end": 5922
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 5931,
                      "end": 5944
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5931,
                    "end": 5944
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5953,
                      "end": 5963
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5953,
                    "end": 5963
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5972,
                      "end": 5981
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5972,
                    "end": 5981
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5990,
                      "end": 5998
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5990,
                    "end": 5998
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6007,
                      "end": 6016
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6007,
                    "end": 6016
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6025,
                      "end": 6029
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
                            "start": 6044,
                            "end": 6046
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6044,
                          "end": 6046
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 6059,
                            "end": 6069
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6059,
                          "end": 6069
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6082,
                            "end": 6091
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6082,
                          "end": 6091
                        }
                      }
                    ],
                    "loc": {
                      "start": 6030,
                      "end": 6101
                    }
                  },
                  "loc": {
                    "start": 6025,
                    "end": 6101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6110,
                      "end": 6122
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
                            "start": 6137,
                            "end": 6139
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6137,
                          "end": 6139
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6152,
                            "end": 6160
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6152,
                          "end": 6160
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6173,
                            "end": 6184
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6173,
                          "end": 6184
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 6197,
                            "end": 6209
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6197,
                          "end": 6209
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6222,
                            "end": 6226
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6222,
                          "end": 6226
                        }
                      }
                    ],
                    "loc": {
                      "start": 6123,
                      "end": 6236
                    }
                  },
                  "loc": {
                    "start": 6110,
                    "end": 6236
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6245,
                      "end": 6257
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6245,
                    "end": 6257
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6266,
                      "end": 6278
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6266,
                    "end": 6278
                  }
                }
              ],
              "loc": {
                "start": 5891,
                "end": 6284
              }
            },
            "loc": {
              "start": 5876,
              "end": 6284
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6289,
                "end": 6291
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6289,
              "end": 6291
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6296,
                "end": 6305
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6296,
              "end": 6305
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6310,
                "end": 6329
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6310,
              "end": 6329
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6334,
                "end": 6349
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6334,
              "end": 6349
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 6354,
                "end": 6363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6354,
              "end": 6363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 6368,
                "end": 6379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6368,
              "end": 6379
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 6384,
                "end": 6395
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6384,
              "end": 6395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 6400,
                "end": 6404
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6400,
              "end": 6404
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 6409,
                "end": 6415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6409,
              "end": 6415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 6420,
                "end": 6430
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6420,
              "end": 6430
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 6435,
                "end": 6446
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6435,
              "end": 6446
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 6451,
                "end": 6470
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6451,
              "end": 6470
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 6475,
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
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6501,
                      "end": 6517
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6498,
                    "end": 6517
                  }
                }
              ],
              "loc": {
                "start": 6488,
                "end": 6523
              }
            },
            "loc": {
              "start": 6475,
              "end": 6523
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 6528,
                "end": 6532
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
                      "start": 6546,
                      "end": 6554
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6543,
                    "end": 6554
                  }
                }
              ],
              "loc": {
                "start": 6533,
                "end": 6560
              }
            },
            "loc": {
              "start": 6528,
              "end": 6560
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6565,
                "end": 6568
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
                      "start": 6579,
                      "end": 6588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6579,
                    "end": 6588
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6597,
                      "end": 6606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6597,
                    "end": 6606
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6615,
                      "end": 6622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6615,
                    "end": 6622
                  }
                }
              ],
              "loc": {
                "start": 6569,
                "end": 6628
              }
            },
            "loc": {
              "start": 6565,
              "end": 6628
            }
          }
        ],
        "loc": {
          "start": 5870,
          "end": 6630
        }
      },
      "loc": {
        "start": 5858,
        "end": 6630
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6631,
          "end": 6633
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6631,
        "end": 6633
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6634,
          "end": 6644
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6634,
        "end": 6644
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6645,
          "end": 6655
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6645,
        "end": 6655
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 6656,
          "end": 6665
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6656,
        "end": 6665
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 6666,
          "end": 6673
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6666,
        "end": 6673
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 6674,
          "end": 6682
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6674,
        "end": 6682
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 6683,
          "end": 6693
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
                "start": 6700,
                "end": 6702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6700,
              "end": 6702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 6707,
                "end": 6724
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6707,
              "end": 6724
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 6729,
                "end": 6741
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6729,
              "end": 6741
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 6746,
                "end": 6756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6746,
              "end": 6756
            }
          }
        ],
        "loc": {
          "start": 6694,
          "end": 6758
        }
      },
      "loc": {
        "start": 6683,
        "end": 6758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 6759,
          "end": 6770
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
                "start": 6777,
                "end": 6779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6777,
              "end": 6779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 6784,
                "end": 6798
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6784,
              "end": 6798
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 6803,
                "end": 6811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6803,
              "end": 6811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 6816,
                "end": 6825
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6816,
              "end": 6825
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 6830,
                "end": 6840
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6830,
              "end": 6840
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 6845,
                "end": 6850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6845,
              "end": 6850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 6855,
                "end": 6862
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6855,
              "end": 6862
            }
          }
        ],
        "loc": {
          "start": 6771,
          "end": 6864
        }
      },
      "loc": {
        "start": 6759,
        "end": 6864
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6894,
          "end": 6896
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6894,
        "end": 6896
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6897,
          "end": 6907
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6897,
        "end": 6907
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 6908,
          "end": 6911
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6908,
        "end": 6911
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6912,
          "end": 6921
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6912,
        "end": 6921
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 6922,
          "end": 6934
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
                "start": 6941,
                "end": 6943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6941,
              "end": 6943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 6948,
                "end": 6956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6948,
              "end": 6956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 6961,
                "end": 6972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6961,
              "end": 6972
            }
          }
        ],
        "loc": {
          "start": 6935,
          "end": 6974
        }
      },
      "loc": {
        "start": 6922,
        "end": 6974
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6975,
          "end": 6978
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
                "start": 6985,
                "end": 6990
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6985,
              "end": 6990
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6995,
                "end": 7007
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6995,
              "end": 7007
            }
          }
        ],
        "loc": {
          "start": 6979,
          "end": 7009
        }
      },
      "loc": {
        "start": 6975,
        "end": 7009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7040,
          "end": 7042
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7040,
        "end": 7042
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7043,
          "end": 7053
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7043,
        "end": 7053
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7054,
          "end": 7064
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7054,
        "end": 7064
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7065,
          "end": 7076
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7065,
        "end": 7076
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7077,
          "end": 7083
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7077,
        "end": 7083
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7084,
          "end": 7089
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7084,
        "end": 7089
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7090,
          "end": 7094
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7090,
        "end": 7094
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7095,
          "end": 7107
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7095,
        "end": 7107
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
                          "value": "pages",
                          "loc": {
                            "start": 349,
                            "end": 354
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
                                  "start": 369,
                                  "end": 371
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 369,
                                "end": 371
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 384,
                                  "end": 393
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 384,
                                "end": 393
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 406,
                                  "end": 410
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 406,
                                "end": 410
                              }
                            }
                          ],
                          "loc": {
                            "start": 355,
                            "end": 420
                          }
                        },
                        "loc": {
                          "start": 349,
                          "end": 420
                        }
                      }
                    ],
                    "loc": {
                      "start": 278,
                      "end": 426
                    }
                  },
                  "loc": {
                    "start": 265,
                    "end": 426
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 431,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 431,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 438,
                      "end": 448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 438,
                    "end": 448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 453,
                      "end": 463
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 453,
                    "end": 463
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 468,
                      "end": 476
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 468,
                    "end": 476
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 481,
                      "end": 490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 481,
                    "end": 490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 495,
                      "end": 507
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 495,
                    "end": 507
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 512,
                      "end": 524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 512,
                    "end": 524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 529,
                      "end": 541
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 529,
                    "end": 541
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 546,
                      "end": 549
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
                            "start": 560,
                            "end": 570
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 560,
                          "end": 570
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 579,
                            "end": 586
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 579,
                          "end": 586
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 595,
                            "end": 604
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 595,
                          "end": 604
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 613,
                            "end": 622
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 613,
                          "end": 622
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 631,
                            "end": 640
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 631,
                          "end": 640
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 649,
                            "end": 655
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 649,
                          "end": 655
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 664,
                            "end": 671
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 664,
                          "end": 671
                        }
                      }
                    ],
                    "loc": {
                      "start": 550,
                      "end": 677
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 677
                  }
                }
              ],
              "loc": {
                "start": 259,
                "end": 679
              }
            },
            "loc": {
              "start": 250,
              "end": 679
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 680,
                "end": 682
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 680,
              "end": 682
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 683,
                "end": 693
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 683,
              "end": 693
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 694,
                "end": 704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 694,
              "end": 704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 705,
                "end": 714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 705,
              "end": 714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 715,
                "end": 726
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 715,
              "end": 726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 727,
                "end": 733
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
                      "start": 743,
                      "end": 753
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 740,
                    "end": 753
                  }
                }
              ],
              "loc": {
                "start": 734,
                "end": 755
              }
            },
            "loc": {
              "start": 727,
              "end": 755
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 756,
                "end": 761
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
                        "start": 775,
                        "end": 787
                      }
                    },
                    "loc": {
                      "start": 775,
                      "end": 787
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
                            "start": 801,
                            "end": 817
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 798,
                          "end": 817
                        }
                      }
                    ],
                    "loc": {
                      "start": 788,
                      "end": 823
                    }
                  },
                  "loc": {
                    "start": 768,
                    "end": 823
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
                        "start": 835,
                        "end": 839
                      }
                    },
                    "loc": {
                      "start": 835,
                      "end": 839
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
                            "start": 853,
                            "end": 861
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 850,
                          "end": 861
                        }
                      }
                    ],
                    "loc": {
                      "start": 840,
                      "end": 867
                    }
                  },
                  "loc": {
                    "start": 828,
                    "end": 867
                  }
                }
              ],
              "loc": {
                "start": 762,
                "end": 869
              }
            },
            "loc": {
              "start": 756,
              "end": 869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 870,
                "end": 881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 870,
              "end": 881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 882,
                "end": 896
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 882,
              "end": 896
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 897,
                "end": 902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 897,
              "end": 902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 903,
                "end": 912
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 903,
              "end": 912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 913,
                "end": 917
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
                      "start": 927,
                      "end": 935
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 924,
                    "end": 935
                  }
                }
              ],
              "loc": {
                "start": 918,
                "end": 937
              }
            },
            "loc": {
              "start": 913,
              "end": 937
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 938,
                "end": 952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 938,
              "end": 952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 953,
                "end": 958
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 953,
              "end": 958
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 959,
                "end": 962
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
                      "start": 969,
                      "end": 978
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 969,
                    "end": 978
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 983,
                      "end": 994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 983,
                    "end": 994
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 999,
                      "end": 1010
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 999,
                    "end": 1010
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1015,
                      "end": 1024
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1015,
                    "end": 1024
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1029,
                      "end": 1036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1029,
                    "end": 1036
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1041,
                      "end": 1049
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1041,
                    "end": 1049
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1054,
                      "end": 1066
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1054,
                    "end": 1066
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1071,
                      "end": 1079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1071,
                    "end": 1079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1084,
                      "end": 1092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1084,
                    "end": 1092
                  }
                }
              ],
              "loc": {
                "start": 963,
                "end": 1094
              }
            },
            "loc": {
              "start": 959,
              "end": 1094
            }
          }
        ],
        "loc": {
          "start": 248,
          "end": 1096
        }
      },
      "loc": {
        "start": 221,
        "end": 1096
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 1106,
          "end": 1122
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 1126,
            "end": 1138
          }
        },
        "loc": {
          "start": 1126,
          "end": 1138
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
                "start": 1141,
                "end": 1143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1141,
              "end": 1143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1144,
                "end": 1155
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1144,
              "end": 1155
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1156,
                "end": 1162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1156,
              "end": 1162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1163,
                "end": 1175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1163,
              "end": 1175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1176,
                "end": 1179
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
                      "start": 1186,
                      "end": 1199
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1186,
                    "end": 1199
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1204,
                      "end": 1213
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1204,
                    "end": 1213
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1218,
                      "end": 1229
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1218,
                    "end": 1229
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1234,
                      "end": 1243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1234,
                    "end": 1243
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1248,
                      "end": 1257
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1248,
                    "end": 1257
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1262,
                      "end": 1269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1262,
                    "end": 1269
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1274,
                      "end": 1286
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1274,
                    "end": 1286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1291,
                      "end": 1299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1291,
                    "end": 1299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1304,
                      "end": 1318
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
                            "start": 1329,
                            "end": 1331
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1329,
                          "end": 1331
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1340,
                            "end": 1350
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1340,
                          "end": 1350
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1359,
                            "end": 1369
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1359,
                          "end": 1369
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1378,
                            "end": 1385
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1378,
                          "end": 1385
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1394,
                            "end": 1405
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1394,
                          "end": 1405
                        }
                      }
                    ],
                    "loc": {
                      "start": 1319,
                      "end": 1411
                    }
                  },
                  "loc": {
                    "start": 1304,
                    "end": 1411
                  }
                }
              ],
              "loc": {
                "start": 1180,
                "end": 1413
              }
            },
            "loc": {
              "start": 1176,
              "end": 1413
            }
          }
        ],
        "loc": {
          "start": 1139,
          "end": 1415
        }
      },
      "loc": {
        "start": 1097,
        "end": 1415
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 1425,
          "end": 1438
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 1442,
            "end": 1450
          }
        },
        "loc": {
          "start": 1442,
          "end": 1450
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
                "start": 1453,
                "end": 1455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1453,
              "end": 1455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1456,
                "end": 1466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1456,
              "end": 1466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1467,
                "end": 1477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1467,
              "end": 1477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1478,
                "end": 1482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1478,
              "end": 1482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1483,
                "end": 1494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1483,
              "end": 1494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1495,
                "end": 1502
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1495,
              "end": 1502
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1503,
                "end": 1508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1503,
              "end": 1508
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1509,
                "end": 1519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1509,
              "end": 1519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 1520,
                "end": 1533
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
                      "start": 1540,
                      "end": 1542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1540,
                    "end": 1542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1547,
                      "end": 1557
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1547,
                    "end": 1557
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1562,
                      "end": 1572
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1562,
                    "end": 1572
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1577,
                      "end": 1581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1577,
                    "end": 1581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1586,
                      "end": 1597
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1586,
                    "end": 1597
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 1602,
                      "end": 1609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1602,
                    "end": 1609
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 1614,
                      "end": 1619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1614,
                    "end": 1619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1624,
                      "end": 1634
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1624,
                    "end": 1634
                  }
                }
              ],
              "loc": {
                "start": 1534,
                "end": 1636
              }
            },
            "loc": {
              "start": 1520,
              "end": 1636
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 1637,
                "end": 1649
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
                      "start": 1656,
                      "end": 1658
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1656,
                    "end": 1658
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1663,
                      "end": 1673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1663,
                    "end": 1673
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1678,
                      "end": 1688
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1678,
                    "end": 1688
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 1693,
                      "end": 1702
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
                            "start": 1713,
                            "end": 1719
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
                                  "start": 1734,
                                  "end": 1736
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1734,
                                "end": 1736
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 1749,
                                  "end": 1754
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1749,
                                "end": 1754
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 1767,
                                  "end": 1772
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1767,
                                "end": 1772
                              }
                            }
                          ],
                          "loc": {
                            "start": 1720,
                            "end": 1782
                          }
                        },
                        "loc": {
                          "start": 1713,
                          "end": 1782
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1791,
                            "end": 1799
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
                                  "start": 1817,
                                  "end": 1832
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1814,
                                "end": 1832
                              }
                            }
                          ],
                          "loc": {
                            "start": 1800,
                            "end": 1842
                          }
                        },
                        "loc": {
                          "start": 1791,
                          "end": 1842
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1851,
                            "end": 1853
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1851,
                          "end": 1853
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1862,
                            "end": 1866
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1862,
                          "end": 1866
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1875,
                            "end": 1886
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1875,
                          "end": 1886
                        }
                      }
                    ],
                    "loc": {
                      "start": 1703,
                      "end": 1892
                    }
                  },
                  "loc": {
                    "start": 1693,
                    "end": 1892
                  }
                }
              ],
              "loc": {
                "start": 1650,
                "end": 1894
              }
            },
            "loc": {
              "start": 1637,
              "end": 1894
            }
          }
        ],
        "loc": {
          "start": 1451,
          "end": 1896
        }
      },
      "loc": {
        "start": 1416,
        "end": 1896
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1906,
          "end": 1919
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1923,
            "end": 1931
          }
        },
        "loc": {
          "start": 1923,
          "end": 1931
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
                "start": 1934,
                "end": 1936
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1934,
              "end": 1936
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1937,
                "end": 1942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1937,
              "end": 1942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1943,
                "end": 1947
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1943,
              "end": 1947
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1948,
                "end": 1955
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1948,
              "end": 1955
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1956,
                "end": 1968
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
                      "start": 1975,
                      "end": 1977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1975,
                    "end": 1977
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1982,
                      "end": 1990
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1982,
                    "end": 1990
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1995,
                      "end": 2006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1995,
                    "end": 2006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2011,
                      "end": 2015
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2011,
                    "end": 2015
                  }
                }
              ],
              "loc": {
                "start": 1969,
                "end": 2017
              }
            },
            "loc": {
              "start": 1956,
              "end": 2017
            }
          }
        ],
        "loc": {
          "start": 1932,
          "end": 2019
        }
      },
      "loc": {
        "start": 1897,
        "end": 2019
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 2029,
          "end": 2044
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2048,
            "end": 2056
          }
        },
        "loc": {
          "start": 2048,
          "end": 2056
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
                "start": 2059,
                "end": 2061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2059,
              "end": 2061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2062,
                "end": 2072
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2062,
              "end": 2072
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2073,
                "end": 2083
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2073,
              "end": 2083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 2084,
                "end": 2093
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2084,
              "end": 2093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2094,
                "end": 2101
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2094,
              "end": 2101
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2102,
                "end": 2110
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2102,
              "end": 2110
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2111,
                "end": 2121
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
                      "start": 2128,
                      "end": 2130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2128,
                    "end": 2130
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2135,
                      "end": 2152
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2135,
                    "end": 2152
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2157,
                      "end": 2169
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2157,
                    "end": 2169
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2174,
                      "end": 2184
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2174,
                    "end": 2184
                  }
                }
              ],
              "loc": {
                "start": 2122,
                "end": 2186
              }
            },
            "loc": {
              "start": 2111,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2187,
                "end": 2198
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
                      "start": 2205,
                      "end": 2207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2205,
                    "end": 2207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2212,
                      "end": 2226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2212,
                    "end": 2226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2231,
                      "end": 2239
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2231,
                    "end": 2239
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2244,
                      "end": 2253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2244,
                    "end": 2253
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2258,
                      "end": 2268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2258,
                    "end": 2268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2273,
                      "end": 2278
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2273,
                    "end": 2278
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2283,
                      "end": 2290
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2283,
                    "end": 2290
                  }
                }
              ],
              "loc": {
                "start": 2199,
                "end": 2292
              }
            },
            "loc": {
              "start": 2187,
              "end": 2292
            }
          }
        ],
        "loc": {
          "start": 2057,
          "end": 2294
        }
      },
      "loc": {
        "start": 2020,
        "end": 2294
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 2304,
          "end": 2317
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2321,
            "end": 2329
          }
        },
        "loc": {
          "start": 2321,
          "end": 2329
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
                "start": 2332,
                "end": 2338
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
                      "start": 2348,
                      "end": 2358
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2345,
                    "end": 2358
                  }
                }
              ],
              "loc": {
                "start": 2339,
                "end": 2360
              }
            },
            "loc": {
              "start": 2332,
              "end": 2360
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 2361,
                "end": 2371
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
                      "start": 2378,
                      "end": 2384
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
                            "start": 2395,
                            "end": 2397
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2395,
                          "end": 2397
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 2406,
                            "end": 2411
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2406,
                          "end": 2411
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2420,
                            "end": 2425
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2420,
                          "end": 2425
                        }
                      }
                    ],
                    "loc": {
                      "start": 2385,
                      "end": 2431
                    }
                  },
                  "loc": {
                    "start": 2378,
                    "end": 2431
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2436,
                      "end": 2438
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2436,
                    "end": 2438
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2443,
                      "end": 2447
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2443,
                    "end": 2447
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2452,
                      "end": 2463
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2452,
                    "end": 2463
                  }
                }
              ],
              "loc": {
                "start": 2372,
                "end": 2465
              }
            },
            "loc": {
              "start": 2361,
              "end": 2465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 2466,
                "end": 2474
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
                      "start": 2481,
                      "end": 2487
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
                            "start": 2501,
                            "end": 2511
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2498,
                          "end": 2511
                        }
                      }
                    ],
                    "loc": {
                      "start": 2488,
                      "end": 2517
                    }
                  },
                  "loc": {
                    "start": 2481,
                    "end": 2517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 2522,
                      "end": 2534
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
                            "start": 2545,
                            "end": 2547
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2545,
                          "end": 2547
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
                          "value": "description",
                          "loc": {
                            "start": 2573,
                            "end": 2584
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2573,
                          "end": 2584
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 2593,
                            "end": 2597
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2593,
                          "end": 2597
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2606,
                            "end": 2610
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2606,
                          "end": 2610
                        }
                      }
                    ],
                    "loc": {
                      "start": 2535,
                      "end": 2616
                    }
                  },
                  "loc": {
                    "start": 2522,
                    "end": 2616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2621,
                      "end": 2623
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2621,
                    "end": 2623
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 2628,
                      "end": 2650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2628,
                    "end": 2650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnOrganizationProfile",
                    "loc": {
                      "start": 2655,
                      "end": 2680
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2655,
                    "end": 2680
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 2685,
                      "end": 2697
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
                            "start": 2708,
                            "end": 2710
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2708,
                          "end": 2710
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 2719,
                            "end": 2730
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2719,
                          "end": 2730
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 2739,
                            "end": 2745
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2739,
                          "end": 2745
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 2754,
                            "end": 2766
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2754,
                          "end": 2766
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 2775,
                            "end": 2778
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
                                  "start": 2793,
                                  "end": 2806
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2793,
                                "end": 2806
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 2819,
                                  "end": 2828
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2819,
                                "end": 2828
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 2841,
                                  "end": 2852
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2841,
                                "end": 2852
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 2865,
                                  "end": 2874
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2865,
                                "end": 2874
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 2887,
                                  "end": 2896
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2887,
                                "end": 2896
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 2909,
                                  "end": 2916
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2909,
                                "end": 2916
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 2929,
                                  "end": 2941
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2929,
                                "end": 2941
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 2954,
                                  "end": 2962
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2954,
                                "end": 2962
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 2975,
                                  "end": 2989
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
                                        "start": 3008,
                                        "end": 3010
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3008,
                                      "end": 3010
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3027,
                                        "end": 3037
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3027,
                                      "end": 3037
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3054,
                                        "end": 3064
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3054,
                                      "end": 3064
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 3081,
                                        "end": 3088
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3081,
                                      "end": 3088
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3105,
                                        "end": 3116
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3105,
                                      "end": 3116
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2990,
                                  "end": 3130
                                }
                              },
                              "loc": {
                                "start": 2975,
                                "end": 3130
                              }
                            }
                          ],
                          "loc": {
                            "start": 2779,
                            "end": 3140
                          }
                        },
                        "loc": {
                          "start": 2775,
                          "end": 3140
                        }
                      }
                    ],
                    "loc": {
                      "start": 2698,
                      "end": 3146
                    }
                  },
                  "loc": {
                    "start": 2685,
                    "end": 3146
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 3151,
                      "end": 3168
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
                            "start": 3179,
                            "end": 3186
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
                                  "start": 3201,
                                  "end": 3203
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3201,
                                "end": 3203
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
                                  "start": 3239,
                                  "end": 3249
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3239,
                                "end": 3249
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 3262,
                                  "end": 3269
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3262,
                                "end": 3269
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 3282,
                                  "end": 3293
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3282,
                                "end": 3293
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 3306,
                                  "end": 3311
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
                                        "start": 3330,
                                        "end": 3332
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3330,
                                      "end": 3332
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3349,
                                        "end": 3359
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3349,
                                      "end": 3359
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3376,
                                        "end": 3386
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3376,
                                      "end": 3386
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3403,
                                        "end": 3407
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3403,
                                      "end": 3407
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 3424,
                                        "end": 3435
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3424,
                                      "end": 3435
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 3452,
                                        "end": 3464
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3452,
                                      "end": 3464
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 3481,
                                        "end": 3493
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
                                              "start": 3516,
                                              "end": 3518
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3516,
                                            "end": 3518
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 3539,
                                              "end": 3550
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3539,
                                            "end": 3550
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 3571,
                                              "end": 3577
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3571,
                                            "end": 3577
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 3598,
                                              "end": 3610
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3598,
                                            "end": 3610
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 3631,
                                              "end": 3634
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
                                                    "start": 3661,
                                                    "end": 3674
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3661,
                                                  "end": 3674
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 3699,
                                                    "end": 3708
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3699,
                                                  "end": 3708
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 3733,
                                                    "end": 3744
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3733,
                                                  "end": 3744
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 3769,
                                                    "end": 3778
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3769,
                                                  "end": 3778
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 3803,
                                                    "end": 3812
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3803,
                                                  "end": 3812
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 3837,
                                                    "end": 3844
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3837,
                                                  "end": 3844
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 3869,
                                                    "end": 3881
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3869,
                                                  "end": 3881
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 3906,
                                                    "end": 3914
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3906,
                                                  "end": 3914
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 3939,
                                                    "end": 3953
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
                                                          "start": 3984,
                                                          "end": 3986
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 3984,
                                                        "end": 3986
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 4015,
                                                          "end": 4025
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4015,
                                                        "end": 4025
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 4054,
                                                          "end": 4064
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4054,
                                                        "end": 4064
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 4093,
                                                          "end": 4100
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4093,
                                                        "end": 4100
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 4129,
                                                          "end": 4140
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 4129,
                                                        "end": 4140
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 3954,
                                                    "end": 4166
                                                  }
                                                },
                                                "loc": {
                                                  "start": 3939,
                                                  "end": 4166
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3635,
                                              "end": 4188
                                            }
                                          },
                                          "loc": {
                                            "start": 3631,
                                            "end": 4188
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3494,
                                        "end": 4206
                                      }
                                    },
                                    "loc": {
                                      "start": 3481,
                                      "end": 4206
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4223,
                                        "end": 4235
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
                                              "start": 4258,
                                              "end": 4260
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4258,
                                            "end": 4260
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4281,
                                              "end": 4289
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4281,
                                            "end": 4289
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4310,
                                              "end": 4321
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4310,
                                            "end": 4321
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4236,
                                        "end": 4339
                                      }
                                    },
                                    "loc": {
                                      "start": 4223,
                                      "end": 4339
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3312,
                                  "end": 4353
                                }
                              },
                              "loc": {
                                "start": 3306,
                                "end": 4353
                              }
                            }
                          ],
                          "loc": {
                            "start": 3187,
                            "end": 4363
                          }
                        },
                        "loc": {
                          "start": 3179,
                          "end": 4363
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4372,
                            "end": 4374
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4372,
                          "end": 4374
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4383,
                            "end": 4393
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4383,
                          "end": 4393
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4402,
                            "end": 4412
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4402,
                          "end": 4412
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4421,
                            "end": 4425
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4421,
                          "end": 4425
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 4434,
                            "end": 4445
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4434,
                          "end": 4445
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 4454,
                            "end": 4466
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4454,
                          "end": 4466
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 4475,
                            "end": 4487
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
                                  "start": 4502,
                                  "end": 4504
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4502,
                                "end": 4504
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 4517,
                                  "end": 4528
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4517,
                                "end": 4528
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 4541,
                                  "end": 4547
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4541,
                                "end": 4547
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 4560,
                                  "end": 4572
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4560,
                                "end": 4572
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4585,
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
                                      "value": "canAddMembers",
                                      "loc": {
                                        "start": 4607,
                                        "end": 4620
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4607,
                                      "end": 4620
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 4637,
                                        "end": 4646
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4637,
                                      "end": 4646
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 4663,
                                        "end": 4674
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4663,
                                      "end": 4674
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 4691,
                                        "end": 4700
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4691,
                                      "end": 4700
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 4717,
                                        "end": 4726
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4717,
                                      "end": 4726
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 4743,
                                        "end": 4750
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4743,
                                      "end": 4750
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 4767,
                                        "end": 4779
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4767,
                                      "end": 4779
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 4796,
                                        "end": 4804
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4796,
                                      "end": 4804
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 4821,
                                        "end": 4835
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
                                              "start": 4858,
                                              "end": 4860
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4858,
                                            "end": 4860
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4881,
                                              "end": 4891
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4881,
                                            "end": 4891
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4912,
                                              "end": 4922
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4912,
                                            "end": 4922
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 4943,
                                              "end": 4950
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4943,
                                            "end": 4950
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 4971,
                                              "end": 4982
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4971,
                                            "end": 4982
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4836,
                                        "end": 5000
                                      }
                                    },
                                    "loc": {
                                      "start": 4821,
                                      "end": 5000
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4589,
                                  "end": 5014
                                }
                              },
                              "loc": {
                                "start": 4585,
                                "end": 5014
                              }
                            }
                          ],
                          "loc": {
                            "start": 4488,
                            "end": 5024
                          }
                        },
                        "loc": {
                          "start": 4475,
                          "end": 5024
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5033,
                            "end": 5045
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
                                  "start": 5060,
                                  "end": 5062
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5060,
                                "end": 5062
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5075,
                                  "end": 5083
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5075,
                                "end": 5083
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5096,
                                  "end": 5107
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5096,
                                "end": 5107
                              }
                            }
                          ],
                          "loc": {
                            "start": 5046,
                            "end": 5117
                          }
                        },
                        "loc": {
                          "start": 5033,
                          "end": 5117
                        }
                      }
                    ],
                    "loc": {
                      "start": 3169,
                      "end": 5123
                    }
                  },
                  "loc": {
                    "start": 3151,
                    "end": 5123
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 5128,
                      "end": 5142
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5128,
                    "end": 5142
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 5147,
                      "end": 5159
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5147,
                    "end": 5159
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5164,
                      "end": 5167
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
                            "start": 5178,
                            "end": 5187
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5178,
                          "end": 5187
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 5196,
                            "end": 5205
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5196,
                          "end": 5205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5214,
                            "end": 5223
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5214,
                          "end": 5223
                        }
                      }
                    ],
                    "loc": {
                      "start": 5168,
                      "end": 5229
                    }
                  },
                  "loc": {
                    "start": 5164,
                    "end": 5229
                  }
                }
              ],
              "loc": {
                "start": 2475,
                "end": 5231
              }
            },
            "loc": {
              "start": 2466,
              "end": 5231
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 5232,
                "end": 5243
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
                      "start": 5250,
                      "end": 5264
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
                            "start": 5275,
                            "end": 5277
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5275,
                          "end": 5277
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5286,
                            "end": 5296
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5286,
                          "end": 5296
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5305,
                            "end": 5313
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5305,
                          "end": 5313
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 5322,
                            "end": 5331
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5322,
                          "end": 5331
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 5340,
                            "end": 5352
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5340,
                          "end": 5352
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 5361,
                            "end": 5373
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5361,
                          "end": 5373
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 5382,
                            "end": 5386
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
                                  "start": 5401,
                                  "end": 5403
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5401,
                                "end": 5403
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 5416,
                                  "end": 5425
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5416,
                                "end": 5425
                              }
                            }
                          ],
                          "loc": {
                            "start": 5387,
                            "end": 5435
                          }
                        },
                        "loc": {
                          "start": 5382,
                          "end": 5435
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 5444,
                            "end": 5456
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
                                  "start": 5471,
                                  "end": 5473
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5471,
                                "end": 5473
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 5486,
                                  "end": 5494
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5486,
                                "end": 5494
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5507,
                                  "end": 5518
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5507,
                                "end": 5518
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 5531,
                                  "end": 5535
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5531,
                                "end": 5535
                              }
                            }
                          ],
                          "loc": {
                            "start": 5457,
                            "end": 5545
                          }
                        },
                        "loc": {
                          "start": 5444,
                          "end": 5545
                        }
                      }
                    ],
                    "loc": {
                      "start": 5265,
                      "end": 5551
                    }
                  },
                  "loc": {
                    "start": 5250,
                    "end": 5551
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5556,
                      "end": 5558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5556,
                    "end": 5558
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5563,
                      "end": 5572
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5563,
                    "end": 5572
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 5577,
                      "end": 5596
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5577,
                    "end": 5596
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 5601,
                      "end": 5616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5601,
                    "end": 5616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 5621,
                      "end": 5630
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5621,
                    "end": 5630
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 5635,
                      "end": 5646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5635,
                    "end": 5646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 5651,
                      "end": 5662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5651,
                    "end": 5662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5667,
                      "end": 5671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5667,
                    "end": 5671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 5676,
                      "end": 5682
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5676,
                    "end": 5682
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 5687,
                      "end": 5697
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5687,
                    "end": 5697
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5702,
                      "end": 5714
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
                            "start": 5728,
                            "end": 5744
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5725,
                          "end": 5744
                        }
                      }
                    ],
                    "loc": {
                      "start": 5715,
                      "end": 5750
                    }
                  },
                  "loc": {
                    "start": 5702,
                    "end": 5750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 5755,
                      "end": 5759
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
                            "start": 5773,
                            "end": 5781
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5770,
                          "end": 5781
                        }
                      }
                    ],
                    "loc": {
                      "start": 5760,
                      "end": 5787
                    }
                  },
                  "loc": {
                    "start": 5755,
                    "end": 5787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5792,
                      "end": 5795
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
                            "start": 5806,
                            "end": 5815
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5806,
                          "end": 5815
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5824,
                            "end": 5833
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5824,
                          "end": 5833
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5842,
                            "end": 5849
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5842,
                          "end": 5849
                        }
                      }
                    ],
                    "loc": {
                      "start": 5796,
                      "end": 5855
                    }
                  },
                  "loc": {
                    "start": 5792,
                    "end": 5855
                  }
                }
              ],
              "loc": {
                "start": 5244,
                "end": 5857
              }
            },
            "loc": {
              "start": 5232,
              "end": 5857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 5858,
                "end": 5869
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
                      "start": 5876,
                      "end": 5890
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
                            "start": 5901,
                            "end": 5903
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5901,
                          "end": 5903
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 5912,
                            "end": 5922
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5912,
                          "end": 5922
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 5931,
                            "end": 5944
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5931,
                          "end": 5944
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 5953,
                            "end": 5963
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5953,
                          "end": 5963
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 5972,
                            "end": 5981
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5972,
                          "end": 5981
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 5990,
                            "end": 5998
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5990,
                          "end": 5998
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6007,
                            "end": 6016
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6007,
                          "end": 6016
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6025,
                            "end": 6029
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
                                  "start": 6044,
                                  "end": 6046
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6044,
                                "end": 6046
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 6059,
                                  "end": 6069
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6059,
                                "end": 6069
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6082,
                                  "end": 6091
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6082,
                                "end": 6091
                              }
                            }
                          ],
                          "loc": {
                            "start": 6030,
                            "end": 6101
                          }
                        },
                        "loc": {
                          "start": 6025,
                          "end": 6101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6110,
                            "end": 6122
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
                                  "start": 6137,
                                  "end": 6139
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6137,
                                "end": 6139
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6152,
                                  "end": 6160
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6152,
                                "end": 6160
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6173,
                                  "end": 6184
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6173,
                                "end": 6184
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 6197,
                                  "end": 6209
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6197,
                                "end": 6209
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6222,
                                  "end": 6226
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6222,
                                "end": 6226
                              }
                            }
                          ],
                          "loc": {
                            "start": 6123,
                            "end": 6236
                          }
                        },
                        "loc": {
                          "start": 6110,
                          "end": 6236
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6245,
                            "end": 6257
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6245,
                          "end": 6257
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6266,
                            "end": 6278
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6266,
                          "end": 6278
                        }
                      }
                    ],
                    "loc": {
                      "start": 5891,
                      "end": 6284
                    }
                  },
                  "loc": {
                    "start": 5876,
                    "end": 6284
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6289,
                      "end": 6291
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6289,
                    "end": 6291
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6296,
                      "end": 6305
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6296,
                    "end": 6305
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6310,
                      "end": 6329
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6310,
                    "end": 6329
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6334,
                      "end": 6349
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6334,
                    "end": 6349
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 6354,
                      "end": 6363
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6354,
                    "end": 6363
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 6368,
                      "end": 6379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6368,
                    "end": 6379
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 6384,
                      "end": 6395
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6384,
                    "end": 6395
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6400,
                      "end": 6404
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6400,
                    "end": 6404
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 6409,
                      "end": 6415
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6409,
                    "end": 6415
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 6420,
                      "end": 6430
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6420,
                    "end": 6430
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 6435,
                      "end": 6446
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6435,
                    "end": 6446
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 6451,
                      "end": 6470
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6451,
                    "end": 6470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 6475,
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
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6501,
                            "end": 6517
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6498,
                          "end": 6517
                        }
                      }
                    ],
                    "loc": {
                      "start": 6488,
                      "end": 6523
                    }
                  },
                  "loc": {
                    "start": 6475,
                    "end": 6523
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 6528,
                      "end": 6532
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
                            "start": 6546,
                            "end": 6554
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6543,
                          "end": 6554
                        }
                      }
                    ],
                    "loc": {
                      "start": 6533,
                      "end": 6560
                    }
                  },
                  "loc": {
                    "start": 6528,
                    "end": 6560
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6565,
                      "end": 6568
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
                            "start": 6579,
                            "end": 6588
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6579,
                          "end": 6588
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6597,
                            "end": 6606
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6597,
                          "end": 6606
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6615,
                            "end": 6622
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6615,
                          "end": 6622
                        }
                      }
                    ],
                    "loc": {
                      "start": 6569,
                      "end": 6628
                    }
                  },
                  "loc": {
                    "start": 6565,
                    "end": 6628
                  }
                }
              ],
              "loc": {
                "start": 5870,
                "end": 6630
              }
            },
            "loc": {
              "start": 5858,
              "end": 6630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6631,
                "end": 6633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6631,
              "end": 6633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6634,
                "end": 6644
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6634,
              "end": 6644
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6645,
                "end": 6655
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6645,
              "end": 6655
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 6656,
                "end": 6665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6656,
              "end": 6665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 6666,
                "end": 6673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6666,
              "end": 6673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 6674,
                "end": 6682
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6674,
              "end": 6682
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 6683,
                "end": 6693
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
                      "start": 6700,
                      "end": 6702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6700,
                    "end": 6702
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 6707,
                      "end": 6724
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6707,
                    "end": 6724
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 6729,
                      "end": 6741
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6729,
                    "end": 6741
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 6746,
                      "end": 6756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6746,
                    "end": 6756
                  }
                }
              ],
              "loc": {
                "start": 6694,
                "end": 6758
              }
            },
            "loc": {
              "start": 6683,
              "end": 6758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 6759,
                "end": 6770
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
                      "start": 6777,
                      "end": 6779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6777,
                    "end": 6779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 6784,
                      "end": 6798
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6784,
                    "end": 6798
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 6803,
                      "end": 6811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6803,
                    "end": 6811
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 6816,
                      "end": 6825
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6816,
                    "end": 6825
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 6830,
                      "end": 6840
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6830,
                    "end": 6840
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 6845,
                      "end": 6850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6845,
                    "end": 6850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 6855,
                      "end": 6862
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6855,
                    "end": 6862
                  }
                }
              ],
              "loc": {
                "start": 6771,
                "end": 6864
              }
            },
            "loc": {
              "start": 6759,
              "end": 6864
            }
          }
        ],
        "loc": {
          "start": 2330,
          "end": 6866
        }
      },
      "loc": {
        "start": 2295,
        "end": 6866
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 6876,
          "end": 6884
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 6888,
            "end": 6891
          }
        },
        "loc": {
          "start": 6888,
          "end": 6891
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
                "start": 6894,
                "end": 6896
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6894,
              "end": 6896
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6897,
                "end": 6907
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6897,
              "end": 6907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 6908,
                "end": 6911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6908,
              "end": 6911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6912,
                "end": 6921
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6912,
              "end": 6921
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 6922,
                "end": 6934
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
                      "start": 6941,
                      "end": 6943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6941,
                    "end": 6943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6948,
                      "end": 6956
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6948,
                    "end": 6956
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6961,
                      "end": 6972
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6961,
                    "end": 6972
                  }
                }
              ],
              "loc": {
                "start": 6935,
                "end": 6974
              }
            },
            "loc": {
              "start": 6922,
              "end": 6974
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6975,
                "end": 6978
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
                      "start": 6985,
                      "end": 6990
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6985,
                    "end": 6990
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6995,
                      "end": 7007
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6995,
                    "end": 7007
                  }
                }
              ],
              "loc": {
                "start": 6979,
                "end": 7009
              }
            },
            "loc": {
              "start": 6975,
              "end": 7009
            }
          }
        ],
        "loc": {
          "start": 6892,
          "end": 7011
        }
      },
      "loc": {
        "start": 6867,
        "end": 7011
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7021,
          "end": 7029
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7033,
            "end": 7037
          }
        },
        "loc": {
          "start": 7033,
          "end": 7037
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
                "start": 7040,
                "end": 7042
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7040,
              "end": 7042
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7043,
                "end": 7053
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7043,
              "end": 7053
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7054,
                "end": 7064
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7054,
              "end": 7064
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7065,
                "end": 7076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7065,
              "end": 7076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7077,
                "end": 7083
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7077,
              "end": 7083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7084,
                "end": 7089
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7084,
              "end": 7089
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7090,
                "end": 7094
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7090,
              "end": 7094
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7095,
                "end": 7107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7095,
              "end": 7107
            }
          }
        ],
        "loc": {
          "start": 7038,
          "end": 7109
        }
      },
      "loc": {
        "start": 7012,
        "end": 7109
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
        "start": 7117,
        "end": 7121
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
              "start": 7123,
              "end": 7128
            }
          },
          "loc": {
            "start": 7122,
            "end": 7128
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
                "start": 7130,
                "end": 7139
              }
            },
            "loc": {
              "start": 7130,
              "end": 7139
            }
          },
          "loc": {
            "start": 7130,
            "end": 7140
          }
        },
        "directives": [],
        "loc": {
          "start": 7122,
          "end": 7140
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
              "start": 7146,
              "end": 7150
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7151,
                  "end": 7156
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7159,
                    "end": 7164
                  }
                },
                "loc": {
                  "start": 7158,
                  "end": 7164
                }
              },
              "loc": {
                "start": 7151,
                "end": 7164
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
                    "start": 7172,
                    "end": 7177
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
                          "start": 7191,
                          "end": 7200
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7188,
                        "end": 7200
                      }
                    }
                  ],
                  "loc": {
                    "start": 7178,
                    "end": 7206
                  }
                },
                "loc": {
                  "start": 7172,
                  "end": 7206
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 7211,
                    "end": 7220
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
                          "start": 7234,
                          "end": 7247
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7231,
                        "end": 7247
                      }
                    }
                  ],
                  "loc": {
                    "start": 7221,
                    "end": 7253
                  }
                },
                "loc": {
                  "start": 7211,
                  "end": 7253
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 7258,
                    "end": 7267
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
                          "start": 7281,
                          "end": 7294
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7278,
                        "end": 7294
                      }
                    }
                  ],
                  "loc": {
                    "start": 7268,
                    "end": 7300
                  }
                },
                "loc": {
                  "start": 7258,
                  "end": 7300
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 7305,
                    "end": 7314
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
                          "start": 7328,
                          "end": 7341
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 7325,
                        "end": 7341
                      }
                    }
                  ],
                  "loc": {
                    "start": 7315,
                    "end": 7347
                  }
                },
                "loc": {
                  "start": 7305,
                  "end": 7347
                }
              }
            ],
            "loc": {
              "start": 7166,
              "end": 7351
            }
          },
          "loc": {
            "start": 7146,
            "end": 7351
          }
        }
      ],
      "loc": {
        "start": 7142,
        "end": 7353
      }
    },
    "loc": {
      "start": 7111,
      "end": 7353
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
